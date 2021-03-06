# Scaffold a Spring Boot project using the Web JSP/JSTL template

## Prerequisites

 - SD tool is installed (>= [0.15.0](https://github.com/snowdrop/spring-boot-cloud-devex/releases/tag/v0.15.0)). See README.md 

## Step by step instructions

- Create a `my-spring-boot` project directory and move under this directory

```bash
cd /path/to/my-spring-boot
```

- Log to your OpenShift's cluster and create a crud namespace

```bash
oc login -u admin -p admin
oc new-project jsp
```

- Create a `MANIFEST`'s project file within the current folder containing the ENV var needed to specify that the format of the archive to be loaded when Spring Boot will start is `*.war` 

```yaml
name: my-spring-boot
env:
  - name: JAVA_APP_JAR
    value: '*.war'
```  

- Scaffold the project using as artifactId - `my-spring-boot` name and the template jsp `jsp`

```bash
sd create -t jsp -i my-spring-boot
```

- Generate the `war` file and push it

```bash
mvn clean package
sd push --mode binary
```

- Start the Spring Boot application

```bash
sd exec start
```

- Open the Openshift's route within your web browser. Execute this request within your terminal to get the host address and next copy/paste it

```bash
oc get route my-spring-boot --template='{{.spec.host}}'
```


 

