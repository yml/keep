# keep

`keep` is a simple password manager that is built on top of `openPGP`. Each account is save in a text file that contains 3 elements separated by a `\n`:

* Password
* Username
* Notes

The filename is the account name.

**Notes :** 
You can stop using keep and and leave with your data when ever you want.
Browse your files (accounts) : 

* `ls ~/.keep/passwords/`

Display contents manually:

* `gpg -d ~/.keep/passwords/example.com`


`Keep` let you manage multiple **profiles**. A Profile is composed of :

* A directory where the passwords are saved. The directory can be shared between users. The username, note and password are safely encrypted but the account name is visible by anyone that has access to the shared folder.
* `RecipientKeysIds` A space separated list of GPG Key Id that the account should be encrypted to.

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
        -r --recipients=KEYS   List of key ids the message should be encypted 
        -d --dir=PATH          Account Directory
        -p --profile=NAME      Profile name
        -c --clipboard         Copy password to the clipboard

```

When you first use `keep` a configuration file is created in `$HOME/.keep/keep.conf`. This JSON file contains the list of profiles:

```
cat ~/.keep/keep.conf 
[
        {
                "Name": "yml",
                "SecringDir": "/home/yml/.gnupg/secring.gpg",
                "PubringDir": "/home/yml/.gnupg/pubring.gpg",
                "AccountDir": "/home/yml/.keep/passwords",
                "RecipientKeyIds": "6A8D785C",
                "SignerKeyID": "6A8D785C"
        },
        {
                "Name": "company",
                "SecringDir": "/home/yml/.gnupg/secring.gpg",
                "PubringDir": "/home/yml/.gnupg/pubring.gpg",
                "AccountDir": "/home/yml/Dropbox/company/secrets/passwords",
                "RecipientKeyIds": "6A8D785C <add the list of space separated key>"
                "SignerKeyID": "6A8D785C"
        }
]
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

*  github.com/atotto/clipboard
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
