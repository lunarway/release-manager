# Release Manager
GitOps release manager for kubernetes configuration reposistories.


## Access to the config repository
The release manager needs read/write permissions to the config repo. 

To create a secret that the release manager can consume: (expects that the filename is identity)

```
kubectl create secret generic release-manager-git-deploy --from-file=identity=key
```

This secret should be mounted to `/etc/release-manager/ssh`