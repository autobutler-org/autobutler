<template>
  <PageContainer>
    <div class="login-container">
      <h1>Login</h1>
      <form class="login-form" @submit.prevent="handleLogin">
        <div class="form-group">
          <label for="email">Email</label>
          <input
            id="email"
            v-model="email"
            type="email"
            required
            placeholder="Enter your email"
          />
        </div>
        <div class="form-group">
          <label for="password">Password</label>
          <input
            id="password"
            v-model="password"
            type="password"
            required
            placeholder="Enter your password"
          />
        </div>
        <button type="submit" class="login-button">Sign In</button>
        <div class="form-footer">
          <a href="#" @click.prevent="forgotPassword">Forgot Password?</a>
          <a href="#" @click.prevent="createAccount">Create Account</a>
        </div>
      </form>
    </div>
  </PageContainer>
</template>

<script setup lang="ts">
import { ref } from "vue";

const email = ref("");
const password = ref("");

const handleLogin = () => {
  // Implement login logic
  console.log("Login attempted", { email: email.value });
};

const forgotPassword = () => {
  // Implement forgot password logic
};

const createAccount = () => {
  // Implement account creation logic
};
</script>

<style scoped>
.login-container {
  max-width: 400px;
  margin: 0 auto;
}

.login-form {
  margin-top: 2rem;
}

.form-group {
  margin-bottom: 1.5rem;
}

label {
  display: block;
  margin-bottom: 0.5rem;
  color: rgba(255, 255, 255, 0.9);
}

input {
  width: 100%;
  padding: 0.8rem;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 4px;
  color: #fff;
  transition: all 0.3s ease;
}

input:focus {
  outline: none;
  border-color: rgba(0, 255, 170, 0.3);
  background: rgba(255, 255, 255, 0.15);
}

.login-button {
  width: 100%;
  padding: 1rem;
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.2),
    rgba(0, 187, 255, 0.2)
  );
  border: 1px solid rgba(0, 255, 170, 0.3);
  border-radius: 4px;
  color: #fff;
  cursor: pointer;
  transition: all 0.3s ease;
}

.login-button:hover {
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.3),
    rgba(0, 187, 255, 0.3)
  );
}

.form-footer {
  margin-top: 1.5rem;
  display: flex;
  justify-content: space-between;
}

.form-footer a {
  color: rgba(255, 255, 255, 0.7);
  text-decoration: none;
  transition: color 0.3s ease;
}

.form-footer a:hover {
  color: #fff;
}
</style>
