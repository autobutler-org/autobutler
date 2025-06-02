<template>
  <PageContainer>
    <div class="features-page">
      <section id="own-intelligence" class="feature-section">
        <div class="feature-content">
          <h2>Own Your Intelligence</h2>
          <p class="feature-description">
            Smart home automation that learns from you, stores nothing in
            corporate clouds.
          </p>
          <div class="feature-details">
            <p>
              Powered by Home Assistant, your smart home becomes truly
              intelligent while maintaining complete privacy. Unlike cloud-based
              solutions that send your data to corporate servers, our system
              processes everything locally.
            </p>
            <p>
              Your home learns your preferences, schedules, and patterns—all
              while keeping that information within your walls. No data is sent
              to third parties, no behavior analysis is shared, and no privacy
              is compromised.
            </p>
          </div>
          <div class="feature-image">
            <!-- Placeholder for Home Assistant dashboard illustration -->
            <div class="image-placeholder">Home Assistant Dashboard</div>
          </div>
        </div>
      </section>

      <section id="subscription-freedom" class="feature-section">
        <div class="feature-content">
          <h2>Break Subscription Chains</h2>
          <p class="feature-description">
            Pay once, own forever. No monthly ransoms, no data hostages.
          </p>
          <div class="feature-details">
            <p>
              Tired of paying monthly fees for services that sell your data? Our
              one-time purchase model gives you complete ownership of your
              digital infrastructure. No recurring charges, no hidden fees, and
              no subscription traps.
            </p>
            <p>
              Your investment in our system is just that—an investment. As we
              improve the software, you get the updates. As we add features,
              they're yours. No more monthly bills, no more service
              interruptions, and no more data held hostage behind paywalls.
            </p>
          </div>
          <div class="feature-image">
            <!-- Placeholder for subscription vs ownership comparison -->
            <div class="image-placeholder">Subscription vs Ownership</div>
          </div>
        </div>
      </section>

      <section id="privacy-design" class="feature-section">
        <div class="feature-content">
          <h2>True Privacy by Design</h2>
          <p class="feature-description">
            Your family calendar, photos, and personal data never leave your
            control.
          </p>
          <div class="feature-details">
            <p>
              Privacy isn't just a feature—it's the foundation of our entire
              system. Every component, from your family calendar to photo
              storage, is designed with privacy as the primary consideration.
              Your data stays on your hardware, in your home, under your
              control.
            </p>
            <p>
              With our VPN solution, you can access your data securely from
              anywhere while maintaining the same level of privacy. Only
              authorized devices on your private network can access your
              information, ensuring your family's data remains protected.
            </p>
          </div>
          <div class="feature-image">
            <!-- Placeholder for privacy architecture diagram -->
            <div class="image-placeholder">Privacy Architecture</div>
          </div>
        </div>
      </section>

      <section id="platform-freedom" class="feature-section">
        <div class="feature-content">
          <h2>Escape Platform Lock-in</h2>
          <p class="feature-description">
            Your data works everywhere, integrates with anything, owned by you
            alone.
          </p>
          <div class="feature-details">
            <p>
              Our system is built on open standards and protocols, ensuring your
              data isn't trapped in proprietary formats. Whether you're using
              iOS, Android, Windows, or Linux, your data remains accessible and
              usable across all your devices.
            </p>
            <p>
              The open architecture means you can integrate with any service or
              device that supports standard protocols. No vendor lock-in, no
              forced ecosystem, and no artificial barriers to your data's
              portability.
            </p>
          </div>
          <div class="feature-image">
            <!-- Placeholder for cross-platform compatibility diagram -->
            <div class="image-placeholder">Cross-Platform Compatibility</div>
          </div>
        </div>
      </section>

      <section id="ad-blocking" class="feature-section">
        <div class="feature-content">
          <h2>Block Ad Surveillance</h2>
          <p class="feature-description">
            Stop every tracker at the DNS level. No ads, no data leaks, no
            corporate spying.
          </p>
          <div class="feature-details">
            <p>
              Our Pi-hole integration blocks ads and trackers at the network
              level, protecting every device in your home. No more targeted ads,
              no more tracking pixels, and no more data collection without your
              consent.
            </p>
            <p>
              The system maintains a comprehensive blocklist of known trackers
              and ad servers, automatically updating to stay ahead of new
              threats. Your browsing habits remain private, and your network
              stays clean of unwanted surveillance.
            </p>
          </div>
          <div class="feature-image">
            <!-- Placeholder for Pi-hole dashboard -->
            <div class="image-placeholder">Pi-hole Dashboard</div>
          </div>
        </div>
      </section>

      <section id="data-vault" class="feature-section">
        <div class="feature-content">
          <h2>Your Personal Data Vault</h2>
          <p class="feature-description">
            Network-attached storage that you physically control. Your files,
            your hardware, your rules.
          </p>
          <div class="feature-details">
            <p>
              Your personal NAS provides secure, high-performance storage for
              all your data. With our self-hosted solution, you maintain
              physical control of your storage hardware while enjoying
              enterprise-grade features like automatic backups, version control,
              and secure sharing.
            </p>
            <p>
              The system includes a local LLM for processing your data, ensuring
              sensitive information never leaves your network. Whether you're
              searching through documents, analyzing data, or generating
              insights, everything happens on your hardware, under your control.
            </p>
          </div>
          <div class="feature-image">
            <!-- Placeholder for NAS and LLM architecture -->
            <div class="image-placeholder">NAS & LLM Architecture</div>
          </div>
        </div>
      </section>
    </div>
  </PageContainer>
</template>

<script setup lang="ts">
// Component logic can be added here if needed
</script>

<style scoped>
.features-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem 0;
}

.feature-section {
  min-height: 100vh;
  padding: 4rem 2rem;
  display: flex;
  align-items: center;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.feature-section:last-child {
  border-bottom: none;
}

.feature-content {
  width: 100%;
  max-width: 1000px;
  margin: 0 auto;
}

h2 {
  font-size: 3rem;
  margin-bottom: 1rem;
  background: linear-gradient(135deg, #ffffff, rgba(255, 255, 255, 0.8));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.feature-description {
  font-size: 1.5rem;
  margin-bottom: 2rem;
  opacity: 0.9;
}

.feature-details {
  margin-bottom: 3rem;
}

.feature-details p {
  font-size: 1.1rem;
  line-height: 1.6;
  margin-bottom: 1rem;
  color: rgba(255, 255, 255, 0.9);
}

.feature-image {
  width: 100%;
  height: 400px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 2rem;
}

.image-placeholder {
  color: rgba(255, 255, 255, 0.3);
  font-size: 1.2rem;
}

@media (max-width: 768px) {
  .feature-section {
    padding: 2rem 1rem;
  }

  h2 {
    font-size: 2rem;
  }

  .feature-description {
    font-size: 1.2rem;
  }

  .feature-details p {
    font-size: 1rem;
  }

  .feature-image {
    height: 300px;
  }
}
</style>
