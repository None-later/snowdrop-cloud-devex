# Go odo controller

- Install the project
  ```bash
  cd $GOPATH/src
  go get github.com/cmoulliard/k8s-supervisor
  cd k8s-supervisor && dep ensure
  ```

- Create `k8s-supervisord` project on OpenShift
  ```bash
  oc new-project k8s-supervisord
  ```
- Export Docker ENV var to access the docker daemon and next login
  ```bash
  eval $(minishift docker-env)
  docker login -u admin -p $(oc whoami -t) $(minishift openshift registry)
  ```
  
- Build the `copy-supervisord` docker image containing the `go supervisord` application

  ```bash
  cd supervisord
  docker build -t $(minishift openshift registry)/k8s-supervisord/copy-supervisord:1.0 -f Dockerfile-copy-supervisord .
  imagebuilder -t $(minishift openshift registry)/k8s-supervisord/copy-supervisord:1.0 .
  ```

- Compile the spring Boot application using maven to package the project as a `uberjar` file

  ```bash
  cd spring-boot
  mvn clean package
  rm -rf target/*-1.0.jar
  ```
  
- Build the docker image of the `Spring Boot Application` and push it to the `OpenShift` docker registry. 
 
  ```bash
  docker build -t $(minishift openshift registry)/k8s-supervisord/spring-boot-http:1.0 . -f Dockerfile-spring-boot
  ```   
  
- Install the `SpringBoot` application without the `initContainer`, shared volume, ...
  ```bash
  oc create -f openshift/spring-boot.yaml
  ```  

- Execute the go program locally to inject the `initContainer`

  **REMARK**: Rename $HOME with the full path to access your `.kube/config` folder

  ```bash
  go run *.go -kubeconfig=$HOME/.kube/config
  Fetching about DC to be injected
  Listing deployments in namespace k8s-supervisord: 
  spring-boot-supervisord
  Updated deployment...
  ```

- Verify if the `initContainer` has been injected within the `DeploymentConfig`

  ```bash
  oc get dc/spring-boot-supervisord -o yaml | grep -A 25 initContainer
  
  initContainers:
  - args:
    - -r
    - /opt/supervisord
    - ' /var/lib/'
    command:
    - /usr/bin/cp
    image: 172.30.1.1:5000/k8s-supervisord/copy-supervisord:1.0
    imagePullPolicy: Always
    name: copy-supervisord
    resources: {}
    terminationMessagePath: /dev/termination-log
    terminationMessagePolicy: File
    volumeMounts:
    - mountPath: /var/lib/supervisord
      name: shared-data
  restartPolicy: Always
  schedulerName: default-scheduler
  securityContext: {}
  terminationGracePeriodSeconds: 30
  volumes:
  - emptyDir: {}
    name: shared-data
  ...
  ```

- Trigger a Deployment by pushing the spring Boot image
  ```bash
  docker push $(minishift openshift registry)/k8s-supervisord/copy-supervisord:1.0
  docker push $(minishift openshift registry)/k8s-supervisord/spring-boot-http:1.0
  ```
  
- Check status of the pod created and next start a command
  ```bash
  SB_POD=$(oc get pods -l app=spring-boot-supervisord -o name)
  SERVICE_IP=$(minishift ip)

  oc rsh $SB_POD /var/lib/supervisord/bin/supervisord ctl status
  echo                             STOPPED   
  run-java                         STOPPED   
  compile-java                     STOPPED   

  oc rsh $SB_POD /var/lib/supervisord/bin/supervisord ctl start run-java
  run-java: started
  
  oc rsh $SB_POD /var/lib/supervisord/bin/supervisord ctl status          
  echo                             STOPPED   
  run-java                         RUNNING   pid 35, uptime 0:00:04
  compile-java                     STOPPED   
  ```  
  