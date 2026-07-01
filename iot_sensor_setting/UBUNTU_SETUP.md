# Ubuntu Setup Guide

This guide installs and runs the thermal analysis desktop tool on Ubuntu.

## Install Python Environment

Install system packages:

```bash
sudo apt update
sudo apt install -y python3 python3-venv python3-pip build-essential libpq-dev
```

Create a virtual environment:

```bash
cd /path/to/iot_sensor_setting
python3 -m venv .venv
source .venv/bin/activate
pip install --upgrade pip
pip install numpy pandas matplotlib PyQt5 SQLAlchemy psycopg2-binary
```

Run the tool:

```bash
source .venv/bin/activate
python process_temperature.py
```

## GUI Notes

This is a PyQt desktop application. Ubuntu needs a graphical session.

Supported options:

- Ubuntu Desktop
- SSH with X11 forwarding: `ssh -X user@server`
- VNC / NoMachine remote desktop

Avoid running this as a headless service because the tool needs an interactive window.
