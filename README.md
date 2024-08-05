### ecr-migrate.

#### migrate a list of aws ecr repositories from one account to another.

<br>

1. download the binary, move to bin directory.
2. before the command all aws role permissions must be configured in both accounts.

**example config.yaml:**

```yaml
repositories:
  - repo/test/app1
  - repo/test/app2
```

**command:**

```
ecr-migrate --from_region="region" --to_region="region" --from="profile" --to="profile" --config_file="config.yaml"
```
