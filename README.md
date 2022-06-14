# Pod-route Operator

## Requirements
operator-sdk : v1.15.0

# Usage
The operator creates the 
- Deployment (will scale up or down to the spec.replicas specified in the podRoute CR, you can specify an container image with spec.image)
- Service
- Route

## Deployment
Run locally
```bash
oc new-project podroute
# Installs the custom resource definitions onto the cluster
make install
# Create the CR on cluster
oc apply -f - <<EOF
---
apiVersion: quay.io/v1alpha1
kind: Podroute
metadata:
  name: test-podroute
  namespace: podroute
spec:
  image: quay.io/austincunningham/always200:latest
  replicas: 3
EOF
# We can then run the operator locally
make run
# Should see something like
2022-06-10T14:41:28.854+0100	INFO	Creating Deployment
2022-06-10T14:41:28.980+0100	INFO	Creating Service
2022-06-10T14:41:29.114+0100	INFO	Creating Route
```


Should work with any REST server provided its container is `EXPOSED 8080`
Change the image in the CR [here](https://github.com/austincunningham/pod-rout/blob/master/config/samples/_v1alpha1_podroute.yaml#L7)

## TODO
- Setup the spec to also take in the port number the container is exposed on