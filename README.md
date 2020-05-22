# Perun mobile bindings
This project provides Android bindings for [go-perun](https://github.com/perun-network/go-perun) called *prnm*.  
Right now it only provides two-party-payment channels.  

## Security Disclaimer
The authors take no responsibility for any loss of digital assets or other damage caused by the use of this software.  
**Do not use this software with real funds**.

### Getting Started
```sh
# Install gomobile, see https://godoc.org/golang.org/x/mobile/cmd/gomobile
go get golang.org/x/mobile/cmd/gomobile
gomobile init
# Get perun-eth-mobile
git clone https://github.com/perun-network/perun-eth-mobile
# Generate the bindings
cd perun-eth-mobile
gomobile bind -o android/app/prnm.aar -target=android
```

The `android/` folder is an Android Studio Project, the only two important files are:  
- `android/app/src/main/java/network/perun/app/MainActivity.java` contains the Apps logic, exemplifying the use of `go-perun`.  
The `MainActivity` uses a `Node` to propose and accept payment channels.
The `Node` is started with a `prnm.Config` which contains all needed configuration for the underlying `prnm.Client`. Its contructor creates a `ProposalHandler` that accepts all incomming channel proposals and forwards the new channels to `Node.accept`. `Node.accept` then starts two threads; one with an `UpdateHandler` that accepts all updates and one as on-chain watcher that reacts to disputes and settling. To propose a channel, `Node.propose` can be used.  
- `android/app/src/main/AndroidManifest.xml` lists the needed App permissions; `INTERNET`,`ACCESS_NETWORK_STATE`,`WRITE_EXTERNAL_STORAGE`,`READ_EXTERNAL_STORAGE`

After importing the `android/` folder in Android Studio, run it in the Emulator or on a real phone.  
The opposite party can be either also an App, or a [perun-eth-demo](https://github.com/perun-network/perun-eth-demo)-node.

## Copyright
Copyright &copy; 2020 Chair of Applied Cryptography, Technische Universität Darmstadt, Germany.
All rights reserved.
Use of the source code is governed by the Apache 2.0 license that can be found in the [LICENSE file](LICENSE).

Contact us at [info@perun.network](mailto:info@perun.network).