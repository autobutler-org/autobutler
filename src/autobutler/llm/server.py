from flask import Flask, request, jsonify
from flask_cors import CORS
import torch
from transformers import AutoModelForCausalLM, AutoTokenizer

app = Flask(__name__)
CORS(app)

# Loading the tokenizer and model
print("Loading TinyLlama tokenizer...")
tokenizer = AutoTokenizer.from_pretrained("TinyLlama/TinyLlama-1.1B-Chat-v1.0")
print("Loading TinyLlama model...")
model = AutoModelForCausalLM.from_pretrained("TinyLlama/TinyLlama-1.1B-Chat-v1.0")
print("Loaded TinyLlama!")

# Using CPU since we're on a Pi
device = torch.device("cpu")
model = model.to(device)


@app.route("/generate", methods=["POST"])
def generate():
    try:
        data = request.get_json()
        message = data.get("message", "")

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
            pad_token_id=tokenizer.eos_token_id,
        )

        response = tokenizer.decode(outputs[0], skip_special_tokens=True)
        response = response.split("<|assistant|>:")[-1].strip()

        return jsonify({"response": response})
    except Exception as e:
        return jsonify({"error": str(e)}), 500


if __name__ == "__main__":
    print("Setting up the api...")
    app.run(host="0.0.0.0", port=8081)
