check-soa
=========

A simple command-line DNS testing tool

Summary
-------

Its main use is for having rapidly an idea of the health of a DNS
zone. It queries each name server of the zone for the SOA record and
displays the value of the serial number for each server.

It tries to be simple and fast. As a result, it is not a competitor
for serious DNS checkers like [ZoneMaster](https://www.zonemaster.fr/).

Usage
-----

    check-soa ZONENAME

`-h` will tell you the possible options.

Installation
------------
It is written in [Go](http://golang.org), so you need a Go compiler 
installed. Here, we assume you have cgo.

check-soa depends on [GoDNS](http://miek.nl/posts/2014/Aug/16/go-dns-package/) so 
you need to install it:
 
    go get github.com/miekg/dns

Then, compile:
    
    make
    
And copy the executable where you want.

Origins
-------
This program is heavily inspired by the original check-soa found in
the excellent book "DNS and BIND" by Liu and Albitz. Unlike the
original one, my check-soa:
* queries the name servers in parallel
* works with IPv4 and IPv6
* works with TCP

Reference site
--------------
[At Github](https://github.com/bortzmeyer/check-soa)

Licence
-------
Copyright (c) 2012, Stephane Bortzmeyer
All rights reserved.

Redistribution and use in source and binary forms, with or without modification,
are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice,
  this list of conditions and the following disclaimer.
* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS 'AS IS'
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
POSSIBILITY OF SUCH DAMAGE.

Author
------

Stephane Bortzmeyer <bortzmeyer@nic.fr>

