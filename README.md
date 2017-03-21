# Radar

## Introduction

Radar is a simple service that monitors docker containers running on a single docker host or swarm cluster. Radar provides service registration for these containers by creating service definitions in an external SkyDNS registry.

## How does it work

Radar will add service definitions in SkyDNS when the containers it detects meet the following requirements:
* Containers are running and healthy
* Containers interface (e.g eth0) is up and configured
* Containers have a hostname configured with FQDN

## How to start
