# ADR-0006: Crash-Only

## Context and Problem Statement

We run our pods in Kubernetes, and they are pretty memory-intensive as some websites have _a lot_ of data to render.
As such, it is not uncommon for our pods to either be `OOMKilled`, or to be killed by the scheduler.

We need to decide on how to handle pods' expected life cycles.

## Considered Options

- Model after striving for clean shutdowns.
- Try to limit crash causes.
- Just let it crash; every pod is ephemeral, and the only way out is crashing.

## Decision Outcome

We're letting it crash:

- We cannot limit crash causes perfectly. Some requests will cause crashes.
- Clean shutdowns do not exist in our clusters; sometimes, pods must crash.
- Crash-only software is a well-documented and proven way to run software,
  e.g. [_Crash-Only Software_, George Candea & Armando Fox, Stanford University](https://dslab.epfl.ch/pubs/crashonly.pdf).
