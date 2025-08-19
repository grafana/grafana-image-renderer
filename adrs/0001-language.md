# ADR-0001: Language of choice

## Context and Problem Statement

We already had a service written in JavaScript & TypeScript run with Node.js.
This has been working fairly well with the features we had.

However, under the new ownership of the service, it is clear that it won't work in the long term:

* The new owners, Grafana Enterprise, is largely Go-first engineers who hire as such.
* The organisation's backend engineers at large primarily know Go rather than Node & TS.
* Other languages would open for more powerful security features that are inherently incompatible with how Node.js v22 works.

## Considered Options

* Do nothing: continue using Node.js.
* Use Go: a native, simple language we already have a pretty vast amount of experience with.
* Use Rust: a native, complex language that is very, very powerful and has a great ecosystem with strong typing, however is hard to hire for and has little adoption internally.
* Use Zig: a very new language with little adoption and a small ecosystem, but is close-to-the-metal, and quite similar to C.

## Decision Outcome

We're going for Go:

* We want to adopt features in Linux such as Linux namespaces, which won't work in Node.js.
* We want to continue the service's maintenance without being burdened by it ourselves; the service should be possible to move around. This excludes languages like Rust and Zig.
