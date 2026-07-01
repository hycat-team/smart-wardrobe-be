# Testing Strategy

Ensure the reliability of the source code when upgrading and maintaining the system.

## 1. Unit Testing

- Write independent tests for each module at the `usecase` or `repository` level.
- Mock external connections (Database, Cache, Third-party AI) using GoMock or Testify.

## 2. How to Run the Test Suite

Run all automated tests on the local environment:

```bash
make test
```

Ensure all test cases pass before submitting a Pull Request.
