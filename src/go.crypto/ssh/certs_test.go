// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssh

import (
	"bytes"
	"crypto/rand"
	"testing"
	"time"
)

// Cert generated by ssh-keygen 6.0p1 Debian-4.
// % ssh-keygen -s ca-key -I test user-key
var exampleSSHCert = `ssh-rsa-cert-v01@openssh.com AAAAHHNzaC1yc2EtY2VydC12MDFAb3BlbnNzaC5jb20AAAAgb1srW/W3ZDjYAO45xLYAwzHBDLsJ4Ux6ICFIkTjb1LEAAAADAQABAAAAYQCkoR51poH0wE8w72cqSB8Sszx+vAhzcMdCO0wqHTj7UNENHWEXGrU0E0UQekD7U+yhkhtoyjbPOVIP7hNa6aRk/ezdh/iUnCIt4Jt1v3Z1h1P+hA4QuYFMHNB+rmjPwAcAAAAAAAAAAAAAAAEAAAAEdGVzdAAAAAAAAAAAAAAAAP//////////AAAAAAAAAIIAAAAVcGVybWl0LVgxMS1mb3J3YXJkaW5nAAAAAAAAABdwZXJtaXQtYWdlbnQtZm9yd2FyZGluZwAAAAAAAAAWcGVybWl0LXBvcnQtZm9yd2FyZGluZwAAAAAAAAAKcGVybWl0LXB0eQAAAAAAAAAOcGVybWl0LXVzZXItcmMAAAAAAAAAAAAAAHcAAAAHc3NoLXJzYQAAAAMBAAEAAABhANFS2kaktpSGc+CcmEKPyw9mJC4nZKxHKTgLVZeaGbFZOvJTNzBspQHdy7Q1uKSfktxpgjZnksiu/tFF9ngyY2KFoc+U88ya95IZUycBGCUbBQ8+bhDtw/icdDGQD5WnUwAAAG8AAAAHc3NoLXJzYQAAAGC8Y9Z2LQKhIhxf52773XaWrXdxP0t3GBVo4A10vUWiYoAGepr6rQIoGGXFxT4B9Gp+nEBJjOwKDXPrAevow0T9ca8gZN+0ykbhSrXLE5Ao48rqr3zP4O1/9P7e6gp0gw8=`

func TestParseCert(t *testing.T) {
	authKeyBytes := []byte(exampleSSHCert)

	key, _, _, rest, err := ParseAuthorizedKey(authKeyBytes)
	if err != nil {
		t.Fatalf("ParseAuthorizedKey: %v", err)
	}
	if len(rest) > 0 {
		t.Errorf("rest: got %q, want empty", rest)
	}

	if _, ok := key.(*Certificate); !ok {
		t.Fatalf("got %#v, want *Certificate", key)
	}

	marshaled := MarshalAuthorizedKey(key)
	// Before comparison, remove the trailing newline that
	// MarshalAuthorizedKey adds.
	marshaled = marshaled[:len(marshaled)-1]
	if !bytes.Equal(authKeyBytes, marshaled) {
		t.Errorf("marshaled certificate does not match original: got %q, want %q", marshaled, authKeyBytes)
	}
}

func TestValidateCert(t *testing.T) {
	key, _, _, _, err := ParseAuthorizedKey([]byte(exampleSSHCert))
	if err != nil {
		t.Fatalf("ParseAuthorizedKey: %v", err)
	}
	validCert, ok := key.(*Certificate)
	if !ok {
		t.Fatalf("got %v (%T), want *Certificate", key, key)
	}
	checker := CertChecker{}
	checker.IsAuthority = func(k PublicKey) bool {
		return bytes.Equal(k.Marshal(), validCert.SignatureKey.Marshal())
	}

	if err := checker.CheckCert("user", validCert); err != nil {
		t.Errorf("Unable to validate certificate: %v", err)
	}
	invalidCert := &Certificate{
		Key:          testPublicKeys["rsa"],
		SignatureKey: testPublicKeys["ecdsa"],
		ValidBefore:  CertTimeInfinity,
		Signature:    &Signature{},
	}
	if err := checker.CheckCert("user", invalidCert); err == nil {
		t.Error("Invalid cert signature passed validation")
	}
}

func TestValidateCertTime(t *testing.T) {
	cert := Certificate{
		ValidPrincipals: []string{"user"},
		Key:             testPublicKeys["rsa"],
		ValidAfter:      50,
		ValidBefore:     100,
	}

	cert.SignCert(rand.Reader, testSigners["ecdsa"])

	for ts, ok := range map[int64]bool{
		25:  false,
		50:  true,
		99:  true,
		100: false,
		125: false,
	} {
		checker := CertChecker{
			Clock: func() time.Time { return time.Unix(ts, 0) },
		}
		checker.IsAuthority = func(k PublicKey) bool {
			return bytes.Equal(k.Marshal(),
				testPublicKeys["ecdsa"].Marshal())
		}

		if v := checker.CheckCert("user", &cert); (v == nil) != ok {
			t.Errorf("Authenticate(%d): %v", ts, v)
		}
	}
}

// TODO(hanwen): tests for
//
// host keys:
// * fallbacks

func TestHostKeyCert(t *testing.T) {
	cert := &Certificate{
		ValidPrincipals: []string{"hostname", "hostname.domain"},
		Key:             testPublicKeys["rsa"],
		ValidBefore:     CertTimeInfinity,
		CertType:        HostCert,
	}
	cert.SignCert(rand.Reader, testSigners["ecdsa"])

	checker := &CertChecker{
		IsAuthority: func(p PublicKey) bool {
			return bytes.Equal(testPublicKeys["ecdsa"].Marshal(), p.Marshal())
		},
	}

	certSigner, err := NewCertSigner(cert, testSigners["rsa"])
	if err != nil {
		t.Errorf("NewCertSigner: %v", err)
	}

	for _, name := range []string{"hostname", "otherhost"} {
		c1, c2, err := netPipe()
		if err != nil {
			t.Fatalf("netPipe: %v", err)
		}
		defer c1.Close()
		defer c2.Close()

		go func() {
			conf := ServerConfig{
				NoClientAuth: true,
			}
			conf.AddHostKey(certSigner)
			_, _, _, err := NewServerConn(c1, &conf)
			if err != nil {
				t.Fatalf("NewServerConn: %v", err)
			}
		}()

		config := &ClientConfig{
			User:            "user",
			HostKeyCallback: checker.CheckHostKey,
		}
		_, _, _, err = NewClientConn(c2, name, config)

		succeed := name == "hostname"
		if (err == nil) != succeed {
			t.Fatalf("NewClientConn(%q): %v", name, err)
		}
	}
}
