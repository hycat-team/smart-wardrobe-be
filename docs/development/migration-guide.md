# Migration Guide

The system uses **Goose** as the tool to manage database updates.

## 1. Core Principles

- Absolutely do not directly edit the existing database scripts in the `/init-db` folder.
- All database schema changes must be done through a new SQL migration file created using Goose.

## 2. Create a new migration file

Run the following command in the terminal to generate a new migration file:

```bash
make migration-create name=migration_file_name
```

This operation will create a new `.sql` file in the `/migrations` folder with a timestamp format and the name you just entered.

## 3. Execute migration

To run all unexecuted migrations on the local database:

```bash
make migration-up
```

To rollback the most recent migration:

```bash
make migration-down
```
