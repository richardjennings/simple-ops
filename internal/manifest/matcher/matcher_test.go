package matcher

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"testing"
)

// examples from https://kubernetes.io/docs/concepts/workloads/
var manifests = []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
    ports:
    - containerPort: 80
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
---
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: frontend
  labels:
    app: guestbook
    tier: frontend
spec:
  # modify replicas according to your case
  replicas: 3
  selector:
    matchLabels:
      tier: frontend
  template:
    metadata:
      labels:
        tier: frontend
    spec:
      containers:
      - name: php-redis
        image: gcr.io/google_samples/gb-frontend:v3
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: web
spec:
  selector:
    matchLabels:
      app: nginx # has to match .spec.template.metadata.labels
  serviceName: "nginx"
  replicas: 3 # by default is 1
  minReadySeconds: 10 # by default is 0
  template:
    metadata:
      labels:
        app: nginx # has to match .spec.selector.matchLabels
    spec:
      terminationGracePeriodSeconds: 10
      containers:
      - name: nginx
        image: k8s.gcr.io/nginx-slim:0.8
        ports:
        - containerPort: 80
          name: web
        volumeMounts:
        - name: www
          mountPath: /usr/share/nginx/html
  volumeClaimTemplates:
  - metadata:
      name: www
    spec:
      accessModes: [ "ReadWriteOnce" ]
      storageClassName: "my-storage-class"
      resources:
        requests:
          storage: 1Gi
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluentd-elasticsearch
  namespace: kube-system
  labels:
    k8s-app: fluentd-logging
spec:
  selector:
    matchLabels:
      name: fluentd-elasticsearch
  template:
    metadata:
      labels:
        name: fluentd-elasticsearch
    spec:
      tolerations:
      # these tolerations are to have the daemonset runnable on control plane nodes
      # remove them if your control plane nodes should not run pods
      - key: node-role.kubernetes.io/control-plane
        operator: Exists
        effect: NoSchedule
      - key: node-role.kubernetes.io/master
        operator: Exists
        effect: NoSchedule
      containers:
      - name: fluentd-elasticsearch
        image: quay.io/fluentd_elasticsearch/fluentd:v2.5.2
        resources:
          limits:
            memory: 200Mi
          requests:
            cpu: 100m
            memory: 200Mi
        volumeMounts:
        - name: varlog
          mountPath: /var/log
        - name: varlibdockercontainers
          mountPath: /var/lib/docker/containers
          readOnly: true
      terminationGracePeriodSeconds: 30
      volumes:
      - name: varlog
        hostPath:
          path: /var/log
      - name: varlibdockercontainers
        hostPath:
          path: /var/lib/docker/containers
---
apiVersion: batch/v1
kind: Job
metadata:
  name: pi
spec:
  template:
    spec:
      containers:
      - name: pi
        image: perl
        command: ["perl",  "-Mbignum=bpi", "-wle", "print bpi(2000)"]
      restartPolicy: Never
  backoffLimit: 4
---
apiVersion: v1
kind: ReplicationController
metadata:
  name: nginx
spec:
  replicas: 3
  selector:
    app: nginx
  template:
    metadata:
      name: nginx
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
        resources:
          limits:
            memory: 20Mi
          requests:
            cpu: 10m
            memory: 10Mi
        ports:
        - containerPort: 80
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: hello
spec:
  schedule: "* * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: hello
            image: busybox:1.28
            imagePullPolicy: IfNotPresent
            command:
            - /bin/sh
            - -c
            - date; echo Hello from the Kubernetes cluster
          restartPolicy: OnFailure
`)

func TestSvc_ListImages_everyImage(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := afero.WriteFile(fs, "/test/manifest.yaml", manifests, 0777); err != nil {
		t.Error(err)
	}
	matches := NewSvc(fs, "/test", logrus.New())
	actual, err := matches.Images("/test/manifest.yaml")
	if err != nil {
		t.Error(err)
	}
	expected := []string{
		"busybox:1.28",
		"nginx:1.14.2",
		"nginx:1.14.2",
		"quay.io/fluentd_elasticsearch/fluentd:v2.5.2",
		"perl",
		"gcr.io/google_samples/gb-frontend:v3",
		"nginx",
		"k8s.gcr.io/nginx-slim:0.8",
	}
	assert.DeepEqual(t, expected, actual)
}

func TestSvc_ListResources(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := afero.WriteFile(fs, "/test/manifest.yaml", manifests, 0777); err != nil {
		t.Error(err)
	}
	matches := NewSvc(fs, "/test", logrus.New())
	actual, err := matches.ContainerResources("/test/manifest.yaml")
	if err != nil {
		t.Error(err)
	}
	expected := ContainerResources{
		{ParentName: "hello", ParentType: "CronJob", Name: "hello"},
		{ParentName: "nginx", ParentType: "Pod", Name: "nginx"},
		{ParentName: "nginx-deployment", ParentType: "Deployment", Name: "nginx"},
		{ParentName: "fluentd-elasticsearch", ParentType: "DaemonSet", Name: "fluentd-elasticsearch", Resource: &ContainerResource{
			Limits:   Conf{Memory: "200Mi", CPU: ""},
			Requests: Conf{Memory: "200Mi", CPU: "100m"},
		}},
		{ParentName: "pi", ParentType: "Job", Name: "pi"},
		{ParentName: "frontend", ParentType: "ReplicaSet", Name: "php-redis"},
		{ParentName: "nginx", ParentType: "ReplicationController", Name: "nginx", Resource: &ContainerResource{
			Limits:   Conf{Memory: "20Mi", CPU: ""},
			Requests: Conf{Memory: "10Mi", CPU: "10m"},
		}},
		{ParentName: "web", ParentType: "StatefulSet", Name: "nginx"},
	}
	assert.DeepEqual(t, expected, actual)
}
