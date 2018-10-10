# SM ToolBox

The SM ToolBox page contains details about all the tools that are needed for instaling / developing the Service Manager Components.

>**Note:** Click on the title of a section to get redirected to the installation instructions.

### [Go (version 1.10+)](https://golang.org/doc/install)

### [GNU Make](https://www.gnu.org/software/make/manual/make.html)

### [Git](https://git-scm.com/)

### [Docker Windows](https://docs.docker.com/docker-for-mac/install/) / [Docker Mac](https://docs.docker.com/docker-for-windows/install/)

### [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

### [Minikube](https://kubernetes.io/docs/getting-started-guides/minikube/#installation)

Alternatively, **Windows** users may follow these steps:

> **Note:** On Windows Minikube versions 26.x does not work on Windows. Currently version 25.2 is the latest tested version on Windows that works properly.

* Setup chocolately package manager

```powershell
    @"%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe" -NoProfile -InputFormat None -ExecutionPolicy Bypass -Command "iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))" && SET "PATH=%PATH%;%ALLUSERSPROFILE%\chocolatey\bin"
```

* Install Minikube

    ```bat
    choco install minikube --version 0.25.2
    ```

>**Note:** In order to reuse the docker daemon and speed up local development check the [Reusing the Docker daemon section of the minikube getting started guide](https://kubernetes.io/docs/getting-started-guides/minikube/#reusing-the-docker-daemon)

#### [Helm](https://github.com/kubernetes/helm/blob/master/docs/install.md)

Alternatively, **Windows** users may follow [these steps](https://medium.com/@JockDaRock/take-the-helm-with-kubernetes-on-windows-c2cd4373104b):


Afterwards, start Minikube and initialize Tiller.

```console
$ minikube start --extra-config=apiserver.Authorization.Mode=RBAC

$ helm init
```
 #### [servicecatalog](https://github.com/kubernetes-incubator/service-catalog/blob/master/docs/install.md)

Alternatively, **Windows** users may follow these steps:

 ```console
 $ kubectl create clusterrolebinding add-on-cluster-admin --clusterrole=cluster-admin --serviceaccount=kube-system:default

 $ helm repo add svc-cat https://svc-catalog-charts.storage.googleapis.com

 $ kubectl create clusterrolebinding tiller-cluster-admin  --clusterrole=cluster-admin --serviceaccount=kube-system:default

 $ kubectl -n kube-system patch deployment tiller-deploy -p '{"spec": {"template": {"spec": {"automountServiceAccountToken": true}}}}'

 $ helm install svc-cat/catalog  --name catalog --namespace catalog
 ```

### [CF CLI and PCF Dev](https://pivotal.io/platform/pcf-tutorials/getting-started-with-pivotal-cloud-foundry-dev/introduction)

The Service Manager components use go1.10 and currenty  the buildpack installed in PCF Dev does not support go1.10. The following steps install the latest go buildpack version on PCF Dev:

* Download the latest Release from [here](https://github.com/cloudfoundry/go-buildpack/releases)
* Rename the zip so it cotnains no dots (.) and slashes (-)
* Navigate to the directory the zip was downloaded to and run:

    ```console
    $ cf update-buildpack go_buildpack <buildpackzipname>.zip 1
    ```

**Note:** Alternatively, one can use [CF Dev](https://github.com/cloudfoundry-incubator/cfdev)

### [smctl](https://github.com/Peripli/service-manager-cli/blob/master/README.md)

**Note:**  kubectl, svcat, helm and smctl binaries should be in your $PATH.
