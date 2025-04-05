#!/bin/bash

echo "Installing system dependencies..."
apt-get update
apt-get install -y python3 python3-pip python3-venv build-essential git wget

echo "Setting up TinyLlama..."
mkdir -p /opt/tinyllama
cd /opt/tinyllama
rm -rf TinyLlama
git clone https://github.com/jzhang38/TinyLlama.git
cd TinyLlama
python3 -m venv venv
source venv/bin/activate
pip install --upgrade pip
pip install -r requirements.txt
pip install torch transformers

echo "Creating Python API server..."
cat > /opt/tinyllama/server.py << 'EOL'
from flask import Flask, request, jsonify
import torch
from transformers import AutoModelForCausalLM, AutoTokenizer
import os

app = Flask(__name__)

# Loading the tokenizer and model
tokenizer = AutoTokenizer.from_pretrained("TinyLlama/TinyLlama-1.1B-Chat-v1.0")
model = AutoModelForCausalLM.from_pretrained("TinyLlama/TinyLlama-1.1B-Chat-v1.0")

# Using CPU since we're on a Pi
device = torch.device('cpu')
model = model.to(device)

@app.route('/generate', methods=['POST'])
def generate():
    try:
        data = request.get_json()
        message = data.get('message', '')
        
        # Format the input
        prompt = f"<|user|>:{message}<|assistant|>:"
        inputs = tokenizer(prompt, return_tensors="pt").to(device)
        
        # Generate response
        outputs = model.generate(
            **inputs,
            max_new_tokens=1024,
            do_sample=True,
            top_p=0.95,
            top_k=50,
            temperature=0.7,
            num_beams=1,
            pad_token_id=tokenizer.eos_token_id
        )
        
        response = tokenizer.decode(outputs[0], skip_special_tokens=True)
        response = response.split("<|assistant|>:")[-1].strip()
        
        return jsonify({"response": response})
    except Exception as e:
        return jsonify({"error": str(e)}), 500

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8081)
EOL

echo "Creating service file..."
cat > /etc/systemd/system/tinyllama.service << 'EOL'
[Unit]
Description=TinyLlama API Server
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/tinyllama
ExecStart=/opt/tinyllama/TinyLlama/venv/bin/python3 /opt/tinyllama/server.py
Restart=always
Environment=PATH=/opt/tinyllama/TinyLlama/venv/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
Environment=PYTHONPATH=/opt/tinyllama/TinyLlama

[Install]
WantedBy=multi-user.target
EOL

echo "Starting service..."
systemctl daemon-reload
systemctl enable tinyllama
systemctl restart tinyllama

echo "Setup complete! TinyLlama server should be running on port 8081" 