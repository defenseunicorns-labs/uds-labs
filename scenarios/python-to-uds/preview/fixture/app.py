from flask import Flask, jsonify, render_template_string
import os

app = Flask(__name__)

HTML = """<!DOCTYPE html><html><head><title>{{ title }}</title></head><body><h1>Hello from UDS!</h1><p>Environment: {{ env }}</p></body></html>"""

@app.route("/")
def index():
    return render_template_string(HTML, title="My UDS App", env=os.getenv("APP_ENV", "development"))

@app.route("/health")
def health():
    return jsonify({"status": "ok"})

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
