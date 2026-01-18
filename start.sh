#!/bin/bash

EMBED_MODEL="bge-m3"
CHAT_MODEL="llama3.2"
INGEST_SCRIPT="ingest_video.py"
DB_FILE="db.json"
PYTHON_EXEC="python3.11" 

GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Initializing Video Moment Search Engine...${NC}"

cleanup() {
    echo -e "\n${RED}🛑 Shutting down services...${NC}"
    if [ -n "$BACKEND_PID" ]; then
        kill $BACKEND_PID 2>/dev/null
    fi
    if [[ "$VIRTUAL_ENV" != "" ]]; then
        deactivate
    fi
    exit
}

trap cleanup SIGINT

if ! command -v ollama &> /dev/null; then
    echo -e "${BLUE}📦 Ollama not found. Installing via Homebrew...${NC}"
    if ! command -v brew &> /dev/null; then
        echo -e "${RED}❌ Homebrew is not installed! Please install it first.${NC}"
        exit 1
    fi
    brew install ollama
else
    echo -e "${GREEN}✅ Ollama is already installed.${NC}"
fi

if ! curl -s http://localhost:11434/api/tags >/dev/null; then
    echo -e "${BLUE}🧠 Starting Ollama Server...${NC}"
    ollama serve &
    echo "   Waiting for Ollama to be ready..."
    until curl -s http://localhost:11434/api/tags >/dev/null; do
        sleep 1
    done
    echo -e "${GREEN}✅ Ollama is running.${NC}"
else
    echo -e "${GREEN}✅ Ollama is already running.${NC}"
fi

echo -e "${BLUE}⬇️  Checking AI Models...${NC}"
ollama pull $EMBED_MODEL
ollama pull $CHAT_MODEL

if [ ! -f "$DB_FILE" ]; then
    echo -e "${BLUE}📂 '$DB_FILE' not found. Starting Data Ingestion...${NC}"
    
    if ! command -v $PYTHON_EXEC &> /dev/null; then
        echo -e "${RED}❌ $PYTHON_EXEC not found. Please install Python 3.11.${NC}"
        exit 1
    fi

    if [ ! -d "venv" ]; then
        echo -e "   📦 Creating Python virtual environment using $PYTHON_EXEC..."
        $PYTHON_EXEC -m venv venv
    fi

    source venv/bin/activate

    if [ -f "requirements.txt" ]; then
        echo "   Installing Python dependencies..."
        pip install -r requirements.txt
    else
        echo -e "${RED}❌ requirements.txt not found!${NC}"
        exit 1
    fi

    echo -e "   Running $INGEST_SCRIPT..."
    python $INGEST_SCRIPT

    if [ ! -f "$DB_FILE" ]; then
        echo -e "${RED}❌ Ingestion failed! '$DB_FILE' was not created.${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ Ingestion Complete. Database created.${NC}"
    
    deactivate 

else
    echo -e "${GREEN}✅ '$DB_FILE' found. Skipping ingestion.${NC}"
fi

echo -e "${BLUE}🐹 Starting Go Backend...${NC}"
if [ ! -d "backend" ]; then
    echo -e "${RED}❌ 'backend' directory not found! Are you in the project root?${NC}"
    exit 1
fi

cd backend
go run main.go &
BACKEND_PID=$!
cd ..

sleep 2

echo -e "${BLUE}⚡ Starting Frontend...${NC}"
if [ ! -d "frontend" ]; then
    echo -e "${RED}❌ 'frontend' directory not found!${NC}"
    kill $BACKEND_PID
    exit 1
fi

cd frontend

if ! command -v bun &> /dev/null; then
    echo -e "${BLUE}🥯 Bun not found. Installing via Homebrew Tap...${NC}"
    if ! command -v brew &> /dev/null; then
        echo -e "${RED}❌ Homebrew is not installed! Cannot install Bun.${NC}"
        kill $BACKEND_PID
        exit 1
    fi
    brew tap oven-sh/bun
    brew install bun
else
    echo -e "${GREEN}✅ Bun is already installed.${NC}"
fi

if [ ! -d "node_modules" ]; then
    echo "   Installing Frontend dependencies..."
    bun install
fi

echo -e "${GREEN}✅ System Ready! Access the app at http://localhost:5173${NC}"
echo -e "${BLUE}(Press CTRL+C to stop everything)${NC}"

bun dev