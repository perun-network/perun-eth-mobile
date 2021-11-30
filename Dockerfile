FROM ubuntu:20.04

ARG ANDROID_NDK_VERSION=r21d
ARG ANDROID_SDK_VERSION=6609375
ARG GO_VERSION=1.16
ARG SDK_PLATFORM=android-30
ARG GRADLE_VERSION=6.7
ARG SDK_TOOLS="platforms;android-29 build-tools;30.0.1 build-tools;29.0.3 emulator patcher;v4 platform-tools"

ENV DEBIAN_FRONTEND=noninteractive
RUN dpkg --add-architecture i386 > /dev/null && \
    apt-get update -y > /dev/null && \
    apt-get install -y --no-install-recommends \
    apt-utils \
    libncurses5:i386 \
    libc6:i386 \
    libstdc++6:i386 \
    lib32gcc1 \
    lib32ncurses6 \
    lib32z1 \
    zlib1g:i386 \
    openjdk-8-jdk \
    git \
    wget \
    unzip \
    socat > /dev/null && \
    rm -rf /var/lib/apt/lists/* 

# Install SDK
ENV ANDROID_SDK_ROOT /opt/android-sdk
WORKDIR /tmp
RUN mkdir -p ${ANDROID_SDK_ROOT}/cmdline-tools && \
    wget -q -O sdk.zip https://dl.google.com/android/repository/commandlinetools-linux-${ANDROID_SDK_VERSION}_latest.zip && \
    unzip -q sdk.zip -d ${ANDROID_SDK_ROOT}/cmdline-tools && \
    rm sdk.zip

# ENV setup
ENV JAVA_HOME /usr/lib/jvm/java-8-openjdk-amd64
ENV PATH ${PATH}:${ANDROID_SDK_ROOT}/cmdline-tools/tools/bin:${ANDROID_SDK_ROOT}/platform-tools:${ANDROID_SDK_ROOT}/emulator
ENV _JAVA_OPTIONS -XX:+UnlockExperimentalVMOptions -XX:+UseCGroupMemoryLimitForHeap

# Accept all licenses
RUN wget -O - https://raw.githubusercontent.com/thyrlian/AndroidSDK/master/android-sdk/license_accepter.sh | bash -s -- $ANDROID_SDK_ROOT > /dev/null

# Install NDK
ENV ANDROID_NDK_HOME /opt/android-ndk
RUN wget -q https://dl.google.com/android/repository/android-ndk-${ANDROID_NDK_VERSION}-linux-x86_64.zip && \
    unzip -q android-ndk-${ANDROID_NDK_VERSION}-linux-x86_64.zip && \
    mkdir -p ${ANDROID_NDK_HOME} && \
    mv ./android-ndk-${ANDROID_NDK_VERSION}/* ${ANDROID_NDK_HOME} && \
    rm android-ndk-${ANDROID_NDK_VERSION}-linux-x86_64.zip

# ENV setup
ENV PATH ${PATH}:${ANDROID_NDK_HOME}
# Need this later for golang
ENV ANDROID_HOME $ANDROID_SDK_ROOT

# Gradle
ARG GRADLE_SDK_URL=https://services.gradle.org/distributions/gradle-${GRADLE_VERSION}-bin.zip
RUN wget -q -O gradle.zip "${GRADLE_SDK_URL}" && \
    unzip -q gradle.zip -d ${ANDROID_SDK_ROOT}  && \
    rm -rf gradle.zip
ENV GRADLE_HOME ${ANDROID_SDK_ROOT}/gradle-${GRADLE_VERSION}
ENV PATH ${GRADLE_HOME}/bin:$PATH

# Install all SDK stuff
RUN sdkmanager ${SDK_TOOLS}

WORKDIR /root/
# Install go
RUN wget -q -O - https://raw.githubusercontent.com/canha/golang-tools-install-script/master/goinstall.sh | \
    bash -s -- --version $GO_VERSION > /dev/null 2>&1
ENV GOROOT=/root/.go
ENV GOPATH=/root/go
ENV PATH=$GOROOT/bin:$GOPATH/bin:$PATH

# Install go-mobile
# Work-around for https://github.com/golang/go/issues/46943
RUN go mod init example.com/m && go mod tidy && \
    go version && \
    GO111MODULE=on  go get golang.org/x/mobile/cmd/... && \ 
    GO111MODULE=on  go mod download golang.org/x/exp && \ 
    GO111MODULE=off go get -d golang.org/x/mobile/cmd/gobind && \
    go mod download golang.org/x/mobile && \
    gomobile init 

# Build bindings
WORKDIR /src
