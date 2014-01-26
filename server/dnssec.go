// Copyright (c) 2013 Erik St. Martin, Brian Ketelsen. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package server

import (
	"github.com/miekg/dns"
	"log"
	"os"
	"time"
)

const origTTL uint32 = 3600

type cache struct {
	// rw lock
	m map[string]*dns.RRSIG
}

// ParseKeyFile read a DNSSEC keyfile as generated by dnssec-keygen or other
// utilities. It add ".key" for the public key and ".private" for the private
// key.
func ParseKeyFile(file string) (*dns.DNSKEY, dns.PrivateKey, error) {
	f, e := os.Open(file + ".key")
	if e != nil {
		return nil, nil, e
	}
	k, e := dns.ReadRR(f, file+".key")
	if e != nil {
		return nil, nil, e
	}
	f, e = os.Open(file + ".private")
	if e != nil {
		return nil, nil, e
	}
	p, e := k.(*dns.DNSKEY).ReadPrivateKey(f, file+".private")
	if e != nil {
		return nil, nil, e
	}
	return k.(*dns.DNSKEY), p, nil
}

// sign signs a message m, it takes care of negative or nodata responses as
// well by synthesising NSEC records. It will also cache the signatures, using
// a hash of the signed data as a key as well as the generated NSEC records.
// We also fake the origin TTL in the signature, because we don't want to 
// throw away signatures when services decide to have longer TTL.
func (s *Server) sign(m *dns.Msg, bufsize uint16) {
	// get RRsets...?
	sig := make([]*dns.RRSIG, 1, 2)
	// only sign the key we have
	println(s.Dnskey.String())
	for _, r := range m.Answer {
		if k, ok := r.(*dns.DNSKEY); ok {
			sig[0] = new(dns.RRSIG)
			sig[0].OrigTtl = origTTL
			sig[0].Labels = 2
			sig[0].Algorithm = s.Dnskey.Algorithm
			sig[0].KeyTag = s.Dnskey.KeyTag()
			sig[0].Inception = uint32(time.Now().Unix())
			sig[0].Expiration = uint32(time.Now().Unix())
			sig[0].TypeCovered = k.Hdr.Rrtype
			sig[0].SignerName = k.Hdr.Name
			sig[0].Hdr.Name = k.Hdr.Name
			sig[0].Hdr.Ttl = origTTL
			sig[0].Hdr.Class = dns.ClassINET
			if e := sig[0].Sign(s.Dnskey, []dns.RR{k}); e != nil {
				log.Printf("Failed to sign: %s\n", e.Error())
			}
		}
	}
	m.Answer = append(m.Answer, sig[0])
	return
}
