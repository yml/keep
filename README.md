# keep

`keep` is a simple password manager that is built on top of `opengp`. Each account is save in a text file that contains 3 elements separated by a `\n`:

* Username
* Password
* Notes

The filename is the account name.

**Notes :** You can ditch kip at any time.  Browse your files: ls ~/.kip/passwords/
Display contents manually: gpg -d ~/.kip/passwords/example.com

## Install

Make sure you have a GnuPG key pair: [GnuPG HOWTO](https://help.ubuntu.com/community/GnuPrivacyGuardHowto). GnuPG is secure, open, multi-platform, and will probably be around forever. Can you say the same thing about the way you store your passwords currently ?
You can **go get** keep to install it in `$GOPATH/bin`

```
go get github.com/yml/keep/cmd/...
```

## Usage

`keep` has 3 main subcommands { read | list | add } that let you manage your passwords.

```
keep --help
keep password manager

Usage:
        keep read [options] <file> [--print]
        keep list [options] [<file>]
        keep add [options]

Options:
        -r --recipients         List of key ids the message should be encypted time_colon
        -d --account-dir        Account Directory
```

## Test

`test_data` contains an armored private key that should be imported in your pubring and secring.

```
gpg --allow-secret-key-import --import test_data/6A8D785C.gpg.asc
```

You can run the test suite with the following command:

```
cd $GOPATH/github.com/yml/keep
GPGKEY=6A8D785C GPGPASSPHRASE=keep go test --race --cover -v .
```

## Credits

`keep` is a liberal reimplementation in GO of [kip](https://github.com/grahamking/kip) developed by Graham King. `kip` is a python wrapper on top of GnuPG where `keep` on the other hand is a native GO implementation on build on top of [github.com/golang/crypto](https://github.com/golang/crypto/).

This project takes advantage of the following "vendored" packages :

*  github.com/docopt/docopt-go            
*  github.com/jcmdev0/gpgagent            
*  golang.org/x/crypto/cast5              
*  golang.org/x/crypto/openpgp            
*  golang.org/x/crypto/openpgp/armor      
*  golang.org/x/crypto/openpgp/elgamal    
*  golang.org/x/crypto/openpgp/errors     
*  golang.org/x/crypto/openpgp/packet     
*  golang.org/x/crypto/openpgp/s2k        
*  golang.org/x/crypto/ssh/terminal
