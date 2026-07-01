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

## Application, Usecase, and Handler Structure

- Mapper functions converting between entity/domain models and DTOs/responses must be placed in the `application/mapper` folder of the respective module. Do not place mappers in the usecase file.
- Within `application/mapper`, it is allowed to split into multiple mapper files by business group to avoid a single file becoming too long.
- Helpers serving only one usecase/usecase file must be extracted into a separate file in the same package, and the filename must have the `_helper.go` suffix.
- Do not create a helper if the function only wraps a simple expression and does not clarify the business meaning. Prefer inlining or reusing utilities in `pkg/utils`.
- Return messages in the presentation handler must be declared as variables/constants at the top of the handler file following the existing pattern, rather than hard-coding directly in each response.
