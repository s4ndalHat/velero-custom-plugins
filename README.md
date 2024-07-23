# Velero custom plugins

[![velero-custom-plugins-test](https://github.com/WRKT/velero-custom-plugins/actions/workflows/test-velero-plugins.yaml/badge.svg?branch=main)](https://github.com/WRKT/velero-custom-plugins/actions/workflows/test-velero-plugins.yaml)

This repository will contain plugins for Velero to solve use case during restoreItemAction.

## Building the plugins
To build the plugins, run

```shell
$ make
```

To build the image, run

```shell
$ make container
```

This builds an image tagged as `velero/velero-plugin-example:main`. If you want to specify a different name or version/tag, run:

```bash
$ IMAGE=your-repo/your-name VERSION=your-version-tag make container 
```

## Deploying the plugin

To deploy your plugin image to an Velero server:

1. Make sure your image is pushed to a registry that is accessible to your cluster's nodes.
2. Run `velero plugin add <registry/image:version>`. Example with a dockerhub image: `velero plugin add velero/velero-plugin-example`.

## Using this plugin
On the cluster, create a ConfigMap with the label ``agoracalyce.io/replace-pattern: RestoreItemAction`` like below:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: replace-pattern-config
  namespace: velero
  labels:
    velero.io/plugin-config: ""
    agoracalyce.io/replace-pattern: RestoreItemAction
data:
  # Example of pattern mappings
  old-pattern: new-pattern
```