SSHDropper
==========
PoC (i.e. not great) worm which spreads via SSH and drops malware.

Features:
* Password auth using one username/password pair
* Compile-time configuration
* Attacks 10.0.0.0/8
* Heartbeat messages via DNS queries
* Drops and runs a payload

This was written to showcase go for a talk entitled "Hands-On Writing Malware
in Go" at BSidesDC2019.  While it would probably work in some real
environments, it is probably not the worm you're looking for.  Move along.

Configuration
-------------
The configurable parameters are all in a fairly large `var` block near the top
of the [source](./sshdropper.go).  Only the username and password will probably
need to be changed.  These can be set at compile-time by using the linker's
`-X` flag, something like
```sh
go build -ldflags "-X main.username=harpo -X main.password=swordfish"
```

Dropping a Payload
------------------
By default, the dropped payload will just send `date`'s output to `/tmp/t`.
This is probably good for a quick (and safeish) PoC.  A better payload can be
added by removing the last line (`const payload = "#!/bin/sh\ndate > /tmp/t"`)
and appending a binary encoded as a string.  A perl snippet is included in the
source to make this a bit easier.  It should be used something like
```sh
ed sshdropper.go
4394
$
const payload = "#!/bin/sh\ndate > /tmp/t"
d
wq
4351
$ perl -E '$"="\\x";$/=\16;say q{const payload = "" +};say qq{\t"\\x@{[unpack"(H2)*"]}` +}for(<>);say qq{\t""}' <./payload >>sshdropper.go
```

The downside to appending several hundred thousand lines of code is that some
editors tend to not do well with really long files.
[`ed(1)`](https://man.openbsd.org/ed.1) can be used to remove the appended
payload.  In fact, this worm was written entirely with `ed(1)`.

Shortcomings
------------
As this worm was meant to demonstrate using Go rather than be an off-the-shelf
solution for malware propagation, it has a few shortcomings:
* It tries random addresses in 10.0.0.0/8.  There's no good way to change this

* It only tries a single username/password pair and only with password auth
* It listens on a set port (31337) to keep multiple instances from running
which is a bit racy
* It has no good way to let anybody know it's done anything except the regular
DNS heartbeats
* It makes regular DNS heartbeats which can be signatured
* It leaves artifacts in /tmp
* It writes logs to stderr
* Linux-only as it relies on /proc/self/exe
