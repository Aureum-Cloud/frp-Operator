## Usage

[Helm](https://helm.sh) must be installed to use the charts. Please refer to
Helm's [documentation](https://helm.sh/docs) to get started.

Once Helm has been set up correctly, add the repo as follows:

    helm repo add aureum-cloud-frp-operator https://frp-operator.aureum.cloud

If you had already added this repo earlier, run `helm repo update` to retrieve
the latest versions of the packages.  You can then run `helm search repo
frp-operator` to see the charts.

To install the frp-operator chart:

    helm install my-frp-operator aureum-cloud-frp-operator/frp-operator

To uninstall the chart:

    helm delete my-frp-operator