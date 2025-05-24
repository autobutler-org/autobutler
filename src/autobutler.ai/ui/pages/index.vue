<template>
  <div class="landing-page">
    <div class="gradient-overlay"></div>
    <div class="hero-section">
      <h1>The New Era of Digital Independence Starts With You.</h1>
      <p class="subtitle">
        Own your data. Control your future. Stop paying subscription ransoms
        forever.
      </p>
      <div class="cta-buttons">
        <NuxtLink to="/about" class="btn primary">Get Started</NuxtLink>
        <NuxtLink to="/features" class="btn secondary">Learn More</NuxtLink>
      </div>
    </div>

    <div class="content-wrapper">
      <div class="features-section">
        <div class="feature-grid">
          <a href="/features#own-intelligence" class="feature-card">
            <h3>Own Your Intelligence</h3>
            <p>
              Smart home automation that learns from you, stores nothing in
              corporate clouds.
            </p>
          </a>
          <a href="/features#subscription-freedom" class="feature-card">
            <h3>Break Subscription Chains</h3>
            <p>Pay once, own forever. No monthly ransoms, no data hostages.</p>
          </a>
          <a href="/features#privacy-design" class="feature-card">
            <h3>True Privacy by Design</h3>
            <p>
              Your family calendar, photos, and personal data never leave your
              control.
            </p>
          </a>
          <a href="/features#platform-freedom" class="feature-card">
            <h3>Escape Platform Lock-in</h3>
            <p>
              Your data works everywhere, integrates with anything, owned by you
              alone.
            </p>
          </a>
          <a href="/features#ad-blocking" class="feature-card">
            <h3>Block Ad Surveillance</h3>
            <p>
              Stop every tracker at the DNS level. No ads, no data leaks, no
              corporate spying.
            </p>
          </a>
          <a href="/features#data-vault" class="feature-card">
            <h3>Your Personal Data Vault</h3>
            <p>
              Network-attached storage that you physically control. Your files,
              your hardware, your rules.
            </p>
          </a>
        </div>
      </div>

      <div class="manifesto-section">
        <h2>The Zero-to-One Truth</h2>
        <p>
          Every tech giant built their empire on a lie: that convenience
          requires surrender. We're building the opposite—a world where you keep
          both your data and your freedom. Where your smart home serves you, not
          shareholders.
        </p>
        <p>This isn't just technology. It's digital sovereignty.</p>
      </div>

      <div class="cta-section">
        <p class="highlight">
          Stop renting your digital life. Start owning it.
        </p>
        <NuxtLink to="/about" class="btn primary">Get Started</NuxtLink>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// Component logic can be added here if needed
</script>

<style>
@import "@/assets/css/common.css";

.landing-page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  background: linear-gradient(135deg, #1a1a1a, #232323);
  padding: 20px;
  position: relative;
  overflow: hidden;
}

.hero-section {
  text-align: center;
  padding: 4rem 2rem;
  max-width: 800px;
  margin: 0 auto;
  width: 100%;
}

.content-wrapper {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
}

.hero-section h1 {
  font-size: 3.5rem;
  line-height: 1.2;
  margin-bottom: 1.5rem;
  background: linear-gradient(135deg, #ffffff, rgba(255, 255, 255, 0.8));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.subtitle {
  font-size: 1.5rem;
  margin-bottom: 2.5rem;
  opacity: 0.9;
}

.cta-buttons {
  display: flex;
  gap: 1rem;
  justify-content: center;
  margin-top: 2rem;
}

.features-section {
  padding: 4rem 2rem;
  background: rgba(255, 255, 255, 0.03);
  width: 100%;
  position: relative;
  z-index: 2;
}

.feature-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 2rem;
  position: relative;
  z-index: 2;
  max-width: 1200px;
  margin: 0 auto;
}

.feature-card {
  padding: 2rem;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  transition: all 0.3s ease;
  text-decoration: none;
  color: inherit;
  display: block;
  cursor: pointer;
  position: relative;
  z-index: 2;
}

.feature-card:hover {
  transform: translateY(-5px);
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.2);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
}

.feature-card h3 {
  color: #fff;
  margin-bottom: 1rem;
  transition: color 0.3s ease;
}

.feature-card:hover h3 {
  color: #66d9ff;
}

.feature-card p {
  color: rgba(255, 255, 255, 0.8);
  transition: color 0.3s ease;
}

.feature-card:hover p {
  color: rgba(255, 255, 255, 0.9);
}

.manifesto-section {
  padding: 4rem 2rem;
  text-align: center;
  max-width: 800px;
  margin: 0 auto;
}

.manifesto-section h2 {
  font-size: 2.5rem;
  margin-bottom: 2rem;
  background: linear-gradient(135deg, #ffffff, rgba(255, 255, 255, 0.8));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.cta-section {
  padding: 4rem 2rem;
  text-align: center;
  background: rgba(255, 255, 255, 0.03);
  width: 100%;
}

.highlight {
  font-size: 2rem;
  font-weight: 600;
  margin-bottom: 2rem;
  background: linear-gradient(135deg, #ffffff, rgba(255, 255, 255, 0.8));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.btn {
  font-size: 1.1rem;
  padding: 1rem 2rem;
  border-radius: 8px;
  transition: all 0.3s ease;
}

.btn.primary {
  background: linear-gradient(
    135deg,
    rgba(255, 255, 255, 0.2),
    rgba(255, 255, 255, 0.1)
  );
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.btn.primary:hover {
  background: linear-gradient(
    135deg,
    rgba(255, 255, 255, 0.3),
    rgba(255, 255, 255, 0.2)
  );
  transform: translateY(-2px);
}

.btn.secondary {
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.btn.secondary:hover {
  background: rgba(255, 255, 255, 0.1);
  transform: translateY(-2px);
}

@media (max-width: 768px) {
  .hero-section h1 {
    font-size: 2.5rem;
  }

  .subtitle {
    font-size: 1.2rem;
  }

  .cta-buttons {
    flex-direction: column;
  }

  .feature-grid {
    grid-template-columns: 1fr;
  }
}
</style>
