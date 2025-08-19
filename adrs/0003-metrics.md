# ADR-0003: Metrics

## Context and Problem Statement

We need a way to communicate metrics with our observability platform.

## Considered Options

* Do nothing: don't serve any metrics, just be a black box that probably works.
* Serve Prometheus metrics for scraping.
* Push Prometheus metrics for scraping.
* Serve OTLP metrics for scraping.
* Push OTLP metrics for scraping.

## Decision Outcome

We're going for serving Prometheus metrics for scraping:

* Not pushing metrics gives us very little insight into how the application is performing, and what's going wrong.
* Prometheus' format is very mature, and the library works very well while also being commonly used.
* Pushing metrics is taking on scope that is already solved by the likes of Grafana Alloy and even Prometheus itself.
* OTLP metrics collectors support Prometheus' format, but Prometheus doesn't necessarily support OTLP's format.
