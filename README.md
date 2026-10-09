# Coordination Standard 0.13.0

This branch keeps Coordination Standard 0.13.0, the earlier standard for any coordinator agent. It is no longer maintained. The current standard is the [Claude-herdr Coordination Standard](https://github.com/minqiyang/coordination-standard) on `main`.

One coordinator hands out task cards, reads the workers' reports, and decides what gets accepted, with review depth set by a lane for each card.

- Policy files: [`coordinator.md`](coordination-standard/coordinator.md), [`routing_table.json`](coordination-standard/routing_table.json), [`model_bindings.json`](coordination-standard/model_bindings.json)
- Start: [`coordination-standard/README.md`](coordination-standard/README.md)

![System architecture](coordination-standard/assets/system_architecture.svg)

![Workflow lifecycle](coordination-standard/assets/workflow_lifecycle.svg)

![Auxiliary components](coordination-standard/assets/auxiliary_components.svg)
