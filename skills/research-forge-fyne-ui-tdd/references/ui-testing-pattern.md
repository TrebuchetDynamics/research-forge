# UI testing pattern

Preferred architecture:

- domain services own behavior;
- CLI and Fyne call the same application layer;
- view models expose state and commands;
- Fyne widgets bind to view models;
- background jobs report progress, cancellation, completion, and errors.

Test examples:

- dashboard shows selected project summary;
- search form validates missing query/source;
- search command updates loading/results/error states;
- screening decision button calls shared service and refreshes queue;
- evidence table refuses to accept an item without source support;
- report export propagates build warnings.
