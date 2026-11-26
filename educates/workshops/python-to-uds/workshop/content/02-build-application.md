# Build the application image

The setup hook copied a small Flask application into `$HOME/myapp`.

```terminal:execute
command: |
  cd "$HOME/myapp"
  find . -maxdepth 2 -type f -print
```

Build it with the session's private Docker daemon:

```terminal:execute
command: |
  cd "$HOME/myapp"
  docker build --tag myapp:dev .
```

Confirm that Docker produced an arm64 image:

```terminal:execute
command: |
  docker image inspect --format '{{.RepoTags}} {{.Architecture}}' myapp:dev
```

Run the checkpoint. It verifies the image architecture and imports the Flask application inside a temporary container.

```examiner:execute-test
name: application-image
title: Verify the arm64 application image
timeout: 30
```
