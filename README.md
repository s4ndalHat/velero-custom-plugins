# Velero custom plugins

![GitHub Workflow Status](https://img.shields.io/github/workflow/status/WRKT/velero-custom-plugins/CI?label=build)

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

## Deploying the plugins

To deploy your plugin image to an Velero server:

1. Make sure your image is pushed to a registry that is accessible to your cluster's nodes.
2. Run `velero plugin add <registry/image:version>`. Example with a dockerhub image: `velero plugin add velero/velero-plugin-example`.

## Using this plugins
TO_DO