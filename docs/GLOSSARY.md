# Glossary

## Agent

An AI coding collaborator operating inside the repository.

## Harness

The repo-level operating system that tells humans and agents how to turn intent into safe work.

## Work Packet

The default unit of work: a single file or folder that captures goal, scope, context map, proof, and evidence for one bounded piece of work.

## Story Packet

A work packet expressed through the `story` surface. This is the default shape for bounded work.

## Intake / Warmup

The warmup and classification step that turns a prompt into a work packet shape before implementation begins.

## Classify

Choose the kind of work and the depth/risk lane.

## Map Context

Choose the smallest set of documents and files that must be loaded before work begins.

## Guardrail

A durable project directive that should shape future agent behavior.

## Harness Delta

A documentation, template, validation, backlog, or decision update that makes future agent work safer or easier.

## Backlog Outcome Loop

The feedback workflow for Harness improvements: record predicted impact when a backlog item is created, then record actual measured outcome when the item is closed so future agents can compare expectation with result.

## Durable Layer

The SQLite database and CLI (`scripts/bin/harness-cli`) that stores operational records (intakes, work packets/stories, decisions, guardrails, backlog items, traces) as structured, queryable data. Policy docs describe how to work; the durable layer stores what happened.

## Work Delta

A repository-facing change that moves the selected work forward: docs, code, tests, findings, checklists, tasks, or evidence.

## Product Delta

A work delta that changes product-facing behavior, such as code, tests, API shape, data model, or product docs.

## Trace

A structured record of what an agent did during a task: actions taken, files read, files changed, decisions made, errors encountered, outcome, and any harness friction discovered.