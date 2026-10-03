# Deferred — {{feature name}}

Work discovered while building {{feature name}} that is **not this feature's job**.
Nothing here blocks closing {{feature name}}; nothing here is a task of
{{feature name}}.

This file exists because an unchecked box in `tasks.md` makes a different claim.
`- [ ]` means *this feature is unfinished*. Work belonging to another feature, or
to the test infrastructure itself, is not this feature's unfinished business, and
recording it as such either keeps a closed feature looking open or pressures the
exit condition into being falsified.

## Rules for this file

- Every entry states **what it is**, **why it is out of scope**, and **what would
  unblock it**. An entry that only names a symptom is not an entry.
- An entry that undermines an already-ticked task's evidence **must say so and name
  that task ID**. Otherwise `[X]` reads as unconditional when it is not.
- If an entry turns out to belong to this feature after all, it moves back into
  `tasks.md` as a real task. **This file is not a parking lot.**
- Report every entry in the skill's final report. Deferring silently defeats the
  purpose of recording it.

## Entries

<!--
Format each entry as:

### D-nn — <short title>

- **Vấn đề**: <what is actually wrong, concretely enough that a reader can reproduce
  or judge it without re-deriving your reasoning>
- **Vì sao ngoài phạm vi**: <why this is not {{feature name}}'s job — not "out of time",
  not "hard", but the actual boundary: another feature's responsibility, the shared
  test harness, a third-party constraint>
- **Gỡ bằng cách nào**: <the concrete unblocking step, and where it should live>
- **Ảnh hưởng tới task đã tick**: <task IDs whose evidence this weakens, or "không">

Delete this comment block once the first real entry exists.
-->