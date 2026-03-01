---
name: Bug Report
about: Report a bug to help us improve
title: '[BUG] '
labels: bug
assignees: ''
---

## Bug Description

A clear and concise description of what the bug is.

## Steps to Reproduce

1. Run command '...'
2. With configuration '...'
3. Perform action '...'
4. See error

## Expected Behavior

A clear and concise description of what you expected to happen.

## Actual Behavior

A clear and concise description of what actually happened.

## Environment

- **OS**: [e.g. TrueNAS SCALE 23.10.1, Ubuntu 22.04]
- **Architecture**: [e.g. amd64, arm64]
- **Version**: [e.g. 1.0.0]
- **Go Version**: [e.g. 1.21.5]

## Configuration

```yaml
# Your configuration file (if relevant)
verbose: false
watch:
  mode: write-complete
  recursive: true
  hook: /usr/local/bin/hook.sh
```

## Logs

```
Paste relevant log output here
```

## Additional Context

Add any other context about the problem here.

## Checklist

- [ ] I have searched existing issues to ensure this is not a duplicate
- [ ] I have provided steps to reproduce the bug
- [ ] I have included relevant logs and configuration
- [ ] I have specified my environment details
