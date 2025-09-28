// Ariadne Test Site JavaScript
// Provides DOM manipulation for testing JavaScript-disabled crawlers

(function () {
  "use strict";

  // Wait for DOM to be ready
  document.addEventListener("DOMContentLoaded", function () {
    console.log("Ariadne Test Site JavaScript loaded");

    // Initialize all test features
    initThemeToggle();
    initSearchFunctionality();
    initInteractiveElements();
    initAccessibilityFeatures();
    initCrawlerTestElements();
  });

  // Theme toggle functionality for dark/light mode testing
  function initThemeToggle() {
    const themeToggle = document.querySelector("[data-theme-toggle]");
    const htmlElement = document.documentElement;

    if (themeToggle) {
      themeToggle.addEventListener("click", function () {
        const currentTheme = htmlElement.getAttribute("data-theme");
        const newTheme = currentTheme === "dark" ? "light" : "dark";

        htmlElement.setAttribute("data-theme", newTheme);
        localStorage.setItem("theme-preference", newTheme);

        // Update button text
        themeToggle.textContent = newTheme === "dark" ? "☀️ Light" : "🌙 Dark";

        console.log("Theme toggled to:", newTheme);
      });
    }

    // Load saved theme preference
    const savedTheme = localStorage.getItem("theme-preference") || "light";
    htmlElement.setAttribute("data-theme", savedTheme);
  }

  // Search functionality for testing form interactions
  function initSearchFunctionality() {
    const searchInputs = document.querySelectorAll(
      'input[type="text"], input[type="search"]'
    );

    searchInputs.forEach((input) => {
      input.addEventListener("input", function (e) {
        const query = e.target.value.toLowerCase();
        console.log("Search query:", query);

        // Simple tag filtering for demonstration
        if (input.placeholder.includes("tags")) {
          filterTags(query);
        }
      });
    });
  }

  // Filter tags based on search query
  function filterTags(query) {
    const tagElements = document.querySelectorAll("[data-tag]");

    tagElements.forEach((element) => {
      const tagName = element.getAttribute("data-tag").toLowerCase();
      const shouldShow = !query || tagName.includes(query);

      element.style.display = shouldShow ? "block" : "none";
    });
  }

  // Initialize interactive elements for crawler testing
  function initInteractiveElements() {
    // Add click handlers for cards that should be clickable
    const clickableCards = document.querySelectorAll(".card[data-clickable]");
    clickableCards.forEach((card) => {
      card.style.cursor = "pointer";
      card.addEventListener("click", function () {
        const link = card.querySelector("a");
        if (link) {
          window.location.href = link.href;
        }
      });
    });

    // Newsletter signup form handling
    const newsletterForm = document.querySelector("#newsletter-form");
    if (newsletterForm) {
      newsletterForm.addEventListener("submit", function (e) {
        e.preventDefault();
        const email = newsletterForm.querySelector('input[type="email"]').value;
        console.log("Newsletter signup:", email);
        alert("Newsletter signup simulated for: " + email);
      });
    }

    // Social sharing buttons
    const shareButtons = document.querySelectorAll("[data-share]");
    shareButtons.forEach((button) => {
      button.addEventListener("click", function (e) {
        e.preventDefault();
        const platform = button.getAttribute("data-share");
        const url = window.location.href;
        const title = document.title;

        console.log("Share clicked:", platform, url, title);

        // Simulate sharing (real implementation would open share dialog)
        alert(`Sharing "${title}" on ${platform}`);
      });
    });
  }

  // Accessibility enhancements
  function initAccessibilityFeatures() {
    // Add skip link functionality
    const skipLink = document.querySelector(".skip-link");
    if (skipLink) {
      skipLink.addEventListener("click", function (e) {
        e.preventDefault();
        const mainContent =
          document.querySelector("main") ||
          document.querySelector("#main-content");
        if (mainContent) {
          mainContent.focus();
          mainContent.scrollIntoView();
        }
      });
    }

    // Enhanced keyboard navigation for cards
    const focusableCards = document.querySelectorAll(".card a");
    focusableCards.forEach((link) => {
      link.addEventListener("keydown", function (e) {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          link.click();
        }
      });
    });

    // ARIA live regions for dynamic content
    const liveRegion = document.createElement("div");
    liveRegion.setAttribute("aria-live", "polite");
    liveRegion.setAttribute("aria-atomic", "true");
    liveRegion.className = "sr-only";
    liveRegion.id = "live-region";
    document.body.appendChild(liveRegion);
  }

  // Crawler-specific test elements
  function initCrawlerTestElements() {
    // Add timestamps for deterministic testing
    const timestampElements = document.querySelectorAll("[data-timestamp]");
    timestampElements.forEach((element) => {
      // Use fixed timestamp for deterministic testing
      element.textContent = new Date("2024-01-15T10:00:00Z").toISOString();
    });

    // Add page metadata for crawler analysis
    const pageMetadata = {
      url: window.location.href,
      title: document.title,
      wordCount: countWords(),
      linkCount: document.querySelectorAll("a").length,
      imageCount: document.querySelectorAll("img").length,
      headingCount: document.querySelectorAll("h1, h2, h3, h4, h5, h6").length,
      timestamp: Date.now(),
    };

    // Store metadata for potential crawler access
    window.pageMetadata = pageMetadata;
    console.log("Page metadata:", pageMetadata);

    // Create hidden metadata element for crawlers that parse DOM
    const metadataElement = document.createElement("script");
    metadataElement.type = "application/ld+json";
    metadataElement.textContent = JSON.stringify({
      "@context": "https://schema.org",
      "@type": "WebPage",
      name: document.title,
      url: window.location.href,
      wordCount: pageMetadata.wordCount,
      dateModified: new Date().toISOString(),
    });
    document.head.appendChild(metadataElement);

    // Test lazy loading simulation
    simulateLazyLoading();

    // Test dynamic content loading
    if (window.location.search.includes("dynamic=true")) {
      loadDynamicContent();
    }
  }

  // Count words in main content for metadata
  function countWords() {
    const mainContent = document.querySelector("main") || document.body;
    const text = mainContent.textContent || mainContent.innerText || "";
    return text
      .trim()
      .split(/\s+/)
      .filter((word) => word.length > 0).length;
  }

  // Simulate lazy loading for testing crawler image handling
  function simulateLazyLoading() {
    const lazyImages = document.querySelectorAll("img[data-src]");

    if ("IntersectionObserver" in window) {
      const imageObserver = new IntersectionObserver((entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            const img = entry.target;
            img.src = img.dataset.src;
            img.removeAttribute("data-src");
            imageObserver.unobserve(img);
            console.log("Lazy loaded image:", img.src);
          }
        });
      });

      lazyImages.forEach((img) => imageObserver.observe(img));
    } else {
      // Fallback for crawlers without IntersectionObserver
      lazyImages.forEach((img) => {
        img.src = img.dataset.src;
        img.removeAttribute("data-src");
      });
    }
  }

  // Load dynamic content for SPA testing
  function loadDynamicContent() {
    console.log("Loading dynamic content...");

    setTimeout(() => {
      const dynamicSection = document.querySelector("#dynamic-content");
      if (dynamicSection) {
        dynamicSection.innerHTML = `
                    <h3>Dynamically Loaded Content</h3>
                    <p>This content was loaded via JavaScript after page load.</p>
                    <p>Crawlers need to execute JavaScript to discover this content.</p>
                    <a href="/dynamic-link" class="dynamic-link">Dynamic Link</a>
                `;

        console.log("Dynamic content loaded");

        // Update live region for screen readers
        const liveRegion = document.getElementById("live-region");
        if (liveRegion) {
          liveRegion.textContent = "Dynamic content has been loaded";
        }
      }
    }, 1000);
  }

  // Export functions for testing
  window.ariaidneTestSite = {
    pageMetadata: function () {
      return window.pageMetadata;
    },
    toggleTheme: function (theme) {
      const htmlElement = document.documentElement;
      htmlElement.setAttribute("data-theme", theme);
      console.log("Theme set to:", theme);
    },
    simulateSearch: function (query) {
      filterTags(query);
      console.log("Search simulated:", query);
    },
  };
})();
