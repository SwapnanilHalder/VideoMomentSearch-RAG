import os
import json
import requests
import yt_dlp
import concurrent.futures
from faster_whisper import WhisperModel
from tqdm import tqdm

BASE_DIR = "." 
CONFIG_DIR = os.path.join(BASE_DIR, "config")
DOWNLOADS_DIR = os.path.join(BASE_DIR, "downloads")
CONFIG_FILE = os.path.join(CONFIG_DIR, "resources.json")
DB_FILE = os.path.join(BASE_DIR, "db.json")

def setup_directories():
    if not os.path.exists(DOWNLOADS_DIR): os.makedirs(DOWNLOADS_DIR)
    if not os.path.exists(CONFIG_DIR): os.makedirs(CONFIG_DIR)
    if not os.path.exists(CONFIG_FILE):
        print(f"⚠️  Missing config file at {CONFIG_FILE}. Create it first.")
        exit(0)

def load_config():
    with open(CONFIG_FILE, 'r') as f:
        return json.load(f)

def download_audio(url):
    with yt_dlp.YoutubeDL({'quiet': True, 'extract_flat': True}) as ydl:
        try:
            info = ydl.extract_info(url, download=False)
            video_id = info['id']
            title = info.get('title', video_id)
            filename = f"{video_id}.mp3"
            filepath = os.path.join(DOWNLOADS_DIR, filename)
        except Exception as e:
            print(f"❌ Error fetching metadata for {url}: {e}")
            return None, None, None

    if os.path.exists(filepath):
        print(f"   ✅ Found cached: {title}")
        return filepath, title, video_id

    print(f"   ⬇️  Downloading: {title}...")
    ydl_opts = {
        'format': 'bestaudio/best',
        'postprocessors': [{'key': 'FFmpegExtractAudio','preferredcodec': 'mp3','preferredquality': '192'}],
        'outtmpl': os.path.join(DOWNLOADS_DIR, '%(id)s.%(ext)s'),
        'quiet': True, 'no_warnings': True
    }
    try:
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            ydl.download([url])
        return filepath, title, video_id
    except Exception as e:
        print(f"   ❌ Download failed: {e}")
        return None, None, None

def transcribe_audio(audio_path, model):
    print(f"   🎙️  Transcribing...")
    
    segments, info = model.transcribe(audio_path, beam_size=1)
    
    transcript_data = []
    
    count = 0
    for segment in segments:
        count += 1
        if count % 10 == 0:
            print(f"      Processing segment at {segment.start:.0f}s...", end='\r')
        
        transcript_data.append({
            "start": segment.start,
            "end": segment.end,
            "text": segment.text.strip()
        })
    
    print(f"      ✅ Transcription Complete ({len(transcript_data)} segments).      ")
    return transcript_data

def chunk_transcript(transcript, window_size=30, step_size=15):
    chunks = []
    if not transcript: return []
    
    total_duration = transcript[-1]['end']
    current_start = 0

    while current_start < total_duration:
        current_end = current_start + window_size
        window_text = []
        seg_start_time = -1
        
        for seg in transcript:
            if seg['start'] < current_end and seg['end'] > current_start:
                
                if seg_start_time == -1: 
                    seg_start_time = max(seg['start'], current_start)
                
                window_text.append(seg['text'])
        
        if window_text:
            chunks.append({
                "start": seg_start_time,
                "end": current_end,
                "text": " ".join(window_text)
            })
        
        current_start += step_size
            
    return chunks

def get_embedding(chunk, config):
    try:
        response = requests.post(config['ollama_url'], json={
            "model": config['embedding_model'],
            "prompt": chunk['text'] 
        })
        
        if response.status_code == 200:
            return {
                "text": chunk['text'],
                "start": chunk['start'],
                "end": chunk['end'],
                "embedding": response.json().get("embedding")
            }
        else:
            return None
            
    except Exception:
        return None

def main():
    setup_directories()
    config = load_config()
    
    print(f"🧠 Loading Faster Whisper (Float32 for M1)...")
    try:
        whisper = WhisperModel("small", device="cpu", compute_type="float32")
    except Exception as e:
        print(f"❌ Failed to load Whisper: {e}")
        return

    print(f"🔌 Connected to Ollama (Model: {config['embedding_model']})")
    
    final_database = []
    videos = config['videos']

    print(f"\n🚀 Starting Ingestion Pipeline for {len(videos)} videos...\n")
    
    for url in videos:
        audio_path, title, video_id = download_audio(url)
        if not audio_path: continue

        raw_transcript = transcribe_audio(audio_path, whisper)
        
        chunks = chunk_transcript(raw_transcript)
        
        print(f"   🔮 Generating embeddings for '{title[:30]}...'")
        
        with concurrent.futures.ThreadPoolExecutor(max_workers=5) as executor:
            futures = [executor.submit(get_embedding, chunk, config) for chunk in chunks]
            
            for future in tqdm(concurrent.futures.as_completed(futures), total=len(chunks), unit="vec", leave=False):
                result = future.result()
                if result and result['embedding']:
                    final_database.append({
                        "video_id": video_id,
                        "video_title": title,
                        "start_time": result['start'],
                        "end_time": result['end'],
                        "text": result['text'],
                        "embedding": result['embedding']
                    })
        
        print(f"   ✅ Finished '{title}'\n")

    if len(final_database) > 0:
        print(f"💾 Saving {len(final_database)} vector records to {DB_FILE}...")
        with open(DB_FILE, "w") as f:
            json.dump(final_database, f)
        print("✅ Done! Database ready.")
    else:
        print("❌ No records were processed. Check connections or video URLs.")

if __name__ == "__main__":
    main()