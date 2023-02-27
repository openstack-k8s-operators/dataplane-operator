Contributing
============

Testing
-------

The tests can be run with the following command:
```bash
make test
```

Contributing to documentation
=============================

## Rendering documentation locally

Install docs build requirements into virtualenv:

```
python3 -m venv local/docs-venv
source local/docs-venv/bin/activate
pip install -r docs/doc_requirements.txt
```

Serve docs site on localhost:

```
mkdocs serve
```

Click the link it outputs. As you save changes to files modified in your editor,
the browser will automatically show the new content.

## Create or edit diagrams

Create a `puml` file inside `docs/diagrams/src`

```
touch docs/diagrams/src/demo.puml
```

Check the PlantUML sintax here: <https://plantuml.com/deployment-diagram>

Serve docs site on localhost:

```
mkdocs serve
```

Add the yielded `svg` into the desired `.md` file
```
![Diagram demo](diagrams/out/demo.svg "Diagram demo")
```
