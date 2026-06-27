# MSCS 632 — Chat Application (Team 3)

## Overview

A local, text-based (command-line) chat application that supports sending, receiving,
and displaying messages between a user and their simulated contacts. The user opens a
conversation with a contact, sends a message, the contact receives it and replies, and
the user can review history, filter by contact, and search by keyword.

The same design is implemented in two languages — **Java** and **Go** — so the versions
can be compared. Full details are in `docs/design.docx`.

## Core requirements

- Simulate multiple contacts who can send and receive messages.
- Store a message history; every message records a unique **id**, the **sender**, the
  **conversation peer**, the **body**, and a **timestamp**.
- **Filter** by contact and **search** by keyword (combinable).

## Repository layout

```
MSCS632_team3/
  README.md            — this file
  .gitignore
  docs/
    design.docx        — full design document (includes the timeline)
  impl-java/           — Java implementation (Member 1)
  impl-go/             — Go implementation (Member 2)
```
