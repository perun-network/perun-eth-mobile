# Development cycle

The most needed commands can be found in the [Makefile](Makefile).

## Running the tests

The tests need two Android emulators, as *gomobile* only supports instrumentation (Emulator) tests.  
You can start the complete setup and the tests in Docker with:
```sh
make docker-test # log output in full.log
```
**⚠️ Running the tests needs a lot of resources.**

## Creating the bindings

The binding file will be generated in `android/app/prnm.aar` and a `android/app/prnm-sources.jar` for debugging.

### Docker

I strongly advice to use Docker:  
```sh
make bind-docker
```

### Without Docker

The [Dockerfile](Dockerfile) contains all the dependencies and versions that you will need to reproduce the bindings.  
Mainly:  
- Android NDK
- Android SDK
- Android SDK Tools
- Golang
- Gradle
- Java 8

You can then use `make bind`.

## Updating the Docker image

Replace `<version>` with the version that you want to create. The versioning should follow *go-perun*.

```sh
docker build -t perunnetwork/prnm-ci:<version> .
docker push perunnetwork/prnm-ci:<version>
# Optional if you want to make it the default image
docker tag perunnetwork/prnm-ci:<version> perunnetwork/prnm-ci:latest
docker push perunnetwork/prnm-ci:latest
```

**⚠️ Building the image needs a lot of resources.**
