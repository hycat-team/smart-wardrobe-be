# Coding Conventions

Ensure consistency and quality of the project's source code.

## Formatting

- Use the standard Go formatting command:
    ```bash
    make fmt
    ```
- Before committing code, run the format and structure check:
    ```bash
    make check
    ```

## Error Handling

- Always return the error at the end of a function's return parameter list and check the error immediately.
- Use the predefined `apperror` structure to return clear error results to the front-end.

## Language Standards

Swagger annotations:

- Vietnamese

Client-facing messages:

- Vietnamese

Logger messages:

- English

Code comments:

- English

Identifiers (function names, variables, structs, interfaces, packages, database tables, columns):

- English
