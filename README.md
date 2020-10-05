# dynpower

Dynpower is a server which provides a simple API to update PowerDNS host entries, so PowerDNS can be used as "dynamic" DNS server.

## Introduction

PowerDNS Authoritative Server is a great DNS server. It is running as dockerized instance on two of my virtual machines, so I have the control over the DNS entries without using third party services. Additionally, I wanted to use PowerDNS as "dynamic DNS" server for services running on my home equipment with changing IP address every 24 hours. Some DNS providers like Hurricane Electric or Digitalocean offer this functionality, so it's easy to integrate this into a FRITZ!Box router or a OPNsense firewall.

It seemed that PowerDNS has two solutions - a [Dynamic DNS Update](https://doc.powerdns.com/authoritative/dnsupdate.html) mechanism according to RFC2136 and their own [HTTP API](https://doc.powerdns.com/authoritative/http-api/zone.html) to manipulate objects like servers and zones. The API is also great, but too overwhelming for a small DNS update request, and not usable from a router or firewall system by sending a single HTTP(S) request. So I googled a bit to search for existing solutions. Some of them used third-party PowerDNS admin tools, but most of them relied on some glue code written in Shell, Perl or PHP which updates the PowerDNS database directly. So it decided to give it a try.

The solution has to meet some requirements, which I didn'd found in the amount of blog articles:

* simple HTTP API to update DNS entry
* authorization by access key valid per domain
* configuration of hosts and domain allowed to update
* items like access keys, hosts, domains should not be hard-coded to keep flexible

It would have been easy to write a PHP script to handle the HTTP request, read and write database items and so on. But wait - when using PHP (or Perl, or Python...) in a Docker image, the image has to contain the whole bunch of runtime infrastructure like a webserver to handle the requests (when not using the "development" webserver integrated in PHP), PHP-FPM and the script itself. In my opinion this was exaggerated for this small task. 

Since I wanted to deal with Golang for quite some time, this task was the perfect way to start. So I created the dynpower server module which handles the HTTP request and updates the database after checking the permissions. After this was running, I thought that would be a nice add-on to manage the configuration by a CLI tool. So dynpower-cli was written, it can list, add and delete domain and host entries.

And thanks to the static binding of Go binaries and Docker multi-stage build, the resulting Docker image [geschke/dynpower](https://hub.docker.com/r/geschke/dynpower) with both dynpower server and dynpower-cli included has a very small footprint of less than 16 MB!

## Requirements

* PowerDNS Authoritative Server with MySQL/MariaDB backend
* access to PowerDNS database server

## Usage

It is recommended to use the dynpower Docker image. It comes with dynpower running as server and the dynpower CLI to manage its database entries.

Coming soon, I promise! Sorry, I'm ugly slow in writing this kind of documentation. ;-)

## Status

Currently dynpower is in early stage, it is running on my systems, but untested in another configuration. So I'll be very thankful if you want to test it! If you detect shortcomings or serious problems, please open an [issue](https://github.com/geschke/dynpower/issues) with your bug report here.
