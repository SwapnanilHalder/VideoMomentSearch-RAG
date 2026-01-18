

---

# VideoMomentSearch-RAG

A semantic search engine for YouTube videos running entirely locally. It uses AI to "watch" videos, understand their content, and allow you to search for specific moments using natural language.

## Tech Stack

* **AI Engine:** [Ollama](https://ollama.com/) (Llama 3.2 & BGE-M3)
* **Ingestion:** Python (Faster-Whisper, yt-dlp)
* **Backend:** Go (Gin)
* **Frontend:** React + Vite (running on Bun)

## Quick Start (Mac/Linux)

> **Note:** If you have Homebrew installed, you can use the automated script.

1. Make the script executable:
```bash
chmod +x start.sh

```


2. Run the startup script:
```bash
./start.sh

```



This script will automatically check for dependencies, install Ollama if missing, download necessary AI models, check for the database (running ingestion if needed), and launch both the Backend and Frontend.

## Manual Installation Guide

If you prefer to set things up step-by-step or are on a different OS, follow the guide below.

### Prerequisites

Ensure you have the following installed:

* **Ollama** (The AI Runner):
```bash
brew install ollama

```


* [Go](https://go.dev/dl/) (v1.21+)
* [Bun](https://bun.sh/) (v1.0+)
* [Python 3.11](https://www.python.org/downloads/release/python-3110/) (**Strict Requirement:** This project does not support Python 3.14)
* **FFmpeg** (Required for audio processing):
```bash
brew install ffmpeg

```



### Step 1: Setup AI Engine (Ollama)

Start the Ollama server in a terminal:

```bash
ollama serve

```

In a separate terminal, pull the required models:

```bash
# The Embedding Model (Vector Search)
ollama pull bge-m3

# The Generation Model (RAG Explanation)
ollama pull llama3.2

```

### Step 2: Data Ingestion (Python)

This step downloads audio from the videos listed in `config/resources.json`, transcribes them, and creates the vector database (`db.json`).

1. Create a virtual environment (optional but recommended):
```bash
python3.11 -m venv venv
source venv/bin/activate

```


2. Install dependencies:
```bash
pip install -r requirements.txt

```


3. Run the ingestion script:
```bash
python ingest_video.py

```



> **Wait for it to finish.** You should see a `db.json` file appear in your root folder.

### Step 3: Start the Backend (Go)

The Go server loads `db.json` into memory and exposes the search API.

1. Navigate to the backend folder:
```bash
cd backend

```


2. Run the server:
```bash
go run main.go

```



You should see the message: `🚀 Server running on http://localhost:8080`

### Step 4: Start the Frontend (Bun + React)

1. Open a new terminal and navigate to the frontend:
```bash
cd frontend

```


2. Install dependencies:
```bash
bun install

```


3. Start the development server:
```bash
bun dev

```



Visit **http://localhost:5173** to use the application!

## Project Structure

```text
.
├── start.sh                 # One-click startup script
├── db.json                  # Generated Vector Database (The "Brain")
├── ingest_video.py          # Python script to download & embed videos
├── requirements.txt         # Python dependencies
├── config/
│   └── resources.json       # List of YouTube URLs to index
├── downloads/               # Cached MP3 files
├── backend/
│   └── main.go              # Go API Server (Search & RAG Logic)
└── frontend/
    ├── src/                 # React Source Code
    └── package.json         # Bun/Node dependencies

```