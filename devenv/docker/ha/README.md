# Grafana Image Rendering High Availability (HA) test setup

A set of docker compose services which together creates a Grafana Image Rendering HA test setup with capability of easily
scaling up/down number of Grafana image renderer instances.

Included services

* Grafana
* Grafana image renderer service
* Mysql - Grafana configuration database
* Prometheus - Monitoring of Grafana and used as data source of provisioned alert rules
* cAdvisor - Docker monitoring
* Nginx - Reverse proxy for Grafana, Grafana image renderer and Prometheus. Enables browsing Grafana/renderer/Prometheus UI using a hostname

## Prerequisites

### Build grafana image renderer docker container

Build a Grafana image renderer docker container and tag it as grafana/grafana-image-renderer:dev.

```bash
$ cd <grafana-image-renderer repo>
$ docker build -t grafana/grafana-image-renderer:dev .
```

### Virtual host names

#### Alternative 1 - Use dnsmasq

```bash
$ sudo apt-get install dnsmasq
$ echo 'address=/loc/127.0.0.1' | sudo tee /etc/dnsmasq.d/dnsmasq-loc.conf > /dev/null
$ sudo /etc/init.d/dnsmasq restart
$ ping whatever.loc
PING whatever.loc (127.0.0.1) 56(84) bytes of data.
64 bytes from localhost (127.0.0.1): icmp_seq=1 ttl=64 time=0.076 ms
--- whatever.loc ping statistics ---
1 packet transmitted, 1 received, 0% packet loss, time 1998ms
```

#### Alternative 2 - Manually update /etc/hosts

Update your `/etc/hosts` to be able to access Grafana and/or Prometheus UI using a hostname.

```bash
$ cat /etc/hosts
127.0.0.1       grafana.loc
127.0.0.1       renderer.loc
127.0.0.1       prometheus.loc
```

## Start services

```bash
$ cd <grafana-image-renderer repo>/devenv/docker/ha
$ docker-compose up -d
```

Browse
* http://grafana.loc/
* http://renderer.loc/
* http://prometheus.loc/

Check for any errors

```bash
$ docker-compose logs | grep error
```

You can also provide environment variables for Grafana and Grafana image renderer docker image versions

```bash
$ GRAFANA_VERSION=6.5.0 RENDERER_VERSION=1.0.7 docker-compose up -d
```

### Scale renderer instances up/down

Scale number of image renderer instances to `<instances>`

```bash
$ docker-compose up --scale renderer=<instances> -d
# for example 3 instances
$ docker-compose up --scale renderer=3 -d
```
