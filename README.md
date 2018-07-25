# redmine-sync

redmine-sync is a command to import/export redmine issues.

## Settings

Set the endpoint and api key to your environment variables.

```console
$ export REDMINE_ENDPOINT=http://localhost:8080/
$ export REDMINE_APIKEY=XXXXXXXXXXXXXXXXXXXXXXXXXX
```

## Commands

### Export

`redmine-sync export` exports issues as csv or yaml.

YAML example:

```console
$ redmine-sync export --format yaml
projects:
- id: 1
  tickets:
  - id: 1
    subject: parent ticket
    assignee: Yuhi Ishikura
    status: New
    done_ratio: 55
    description: foo
    tracker: Docs
    start_date: "2018-07-17"
    due_date: "2018-10-20"
    priority: High
    children:
    - id: 20
      subject: doc1
      assignee: null
      status: New
      done_ratio: 10
      description: doc1
...
```

CSV example:

```bash
$ redmine-sync export --format csv
Project,ID,Parent ID,Subject,Assignee,Status,Done Ratio,Description,Tracker,Start Date,Due Date,Priority
aaaa,28,0,ticket20,Redmine Admin,New,0,hogehoge,Bug,2018-07-23,,Normal
aaaa,34,0,ticket4,,New,0,,Bug,2018-07-23,,Normal
aaaa,36,0,ticket6,,New,0,,Bug,2018-07-23,,Normal
aaaa,37,0,test,,New,0,,Bug,2018-07-25,,Normal
...
```

### Import

`redmine-sync import` imports issues with the file.

```console
$ redmine-sync export --format yaml > issues.yml
$ vi issues.yml
$ redmine-sync import issues.yml
```

### Watch

`redmine-sync watch` watch the file modification and automatically import the updates.

```console
$ redmine-sync export --format yaml > issues.yml
$ redmine-sync watch issues.yml
```