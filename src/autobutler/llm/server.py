import torch
from flask import Flask, jsonify, request
from flask_cors import CORS
from transformers import AutoModelForCausalLM, AutoTokenizer

app = Flask(__name__)
CORS(app)

print("Loading TinyLlama tokenizer...")
tokenizer = AutoTokenizer.from_pretrained("TinyLlama/TinyLlama-1.1B-Chat-v1.0")
print("Loading TinyLlama model...")
model = AutoModelForCausalLM.from_pretrained("TinyLlama/TinyLlama-1.1B-Chat-v1.0")
print("Loaded TinyLlama!")

device = torch.device('mps' if torch.backends.mps.is_available() else 'cpu')
print(f"Using device: {device}")
model = model.to(device)
print("Model loaded and ready!")

@app.route('/hello', methods=['GET'])
def hello_world():
    return jsonify({"response": "Hello, world!"})

@app.route('/chat', methods=['POST'])
def chat():
    try:
        data = request.get_json()
        message = data.get('message', '')
        print("no cap ong fr fr")
        return jsonify({"response": f"You said: {message}"})
    except Exception as e:
        return jsonify({"error": str(e)}), 500

@app.route('/test', methods=['GET'])
def test():
    try:
        # Simple test to ensure model is loaded
        test_input = tokenizer("Hello", return_tensors="pt").to(device)
        output = model.generate(**test_input, max_new_tokens=5)
        result = tokenizer.decode(output[0], skip_special_tokens=True)
        return jsonify({
            "status": "Model loaded successfully",
            "device": str(device),
            "test_output": result
        })
    except Exception as e:
        return jsonify({"error": str(e)}), 500

@app.route('/generate', methods=['POST'])
def generate():
    print("Starting /generate endpoint")
    try:
        print("Getting JSON data...")
        data = request.get_json()
        
        if not data:
            print("No JSON data provided")
            return jsonify({"error": "No JSON data provided"}), 400
            
        message = data.get('message', '')
        
        if not message:
            print("No message in JSON data")
            return jsonify({"error": "No message provided"}), 400
            
        print(f"Received message: {message}")
        
        messages = [
            {
                "role": "system",
                "content": "You are a helpful assistant."
            },
            {
                "role": "user", 
                "content": message
            }
        ]
        
        print("Applying chat template...")
        prompt = tokenizer.apply_chat_template(messages, tokenize=False, add_generation_prompt=True)
        print(f"Formatted prompt: {prompt}")
        
        print("Tokenizing input...")
        inputs = tokenizer(prompt, return_tensors="pt").to(device)
        print(f"Input shape: {inputs.input_ids.shape}")
        
        print("Starting generation...")
        outputs = model.generate(
            **inputs,
            max_new_tokens=128,  # Reduced from 256 for faster responses
            do_sample=False,     # Disable sampling for faster generation
            num_beams=1,         # Keep single beam for speed
            pad_token_id=tokenizer.eos_token_id,
            eos_token_id=tokenizer.eos_token_id
        )
        print("Generation complete")
        
        print("Decoding response...")
        response = tokenizer.decode(outputs[0], skip_special_tokens=True)
        print(f"Full decoded response: {response}")
        
        # Extract assistant response
        if "<|assistant|>" in response:
            assistant_response = response.split("<|assistant|>")[-1].strip()
        else:
            # Fallback: remove the prompt from the beginning
            assistant_response = response[len(prompt):].strip()
        
        print(f"Extracted assistant response: {assistant_response}")
        
        return jsonify({"response": assistant_response})
    except Exception as e:
        print(f"Error in /generate endpoint: {str(e)}")
        import traceback
        traceback.print_exc()
        return jsonify({"error": str(e)}), 500


if __name__ == "__main__":
    print("Setting up the api...")
    app.config['SEND_FILE_MAX_AGE_DEFAULT'] = 0
    app.run(host='0.0.0.0', port=8081, threaded=True)
    print("api running")
