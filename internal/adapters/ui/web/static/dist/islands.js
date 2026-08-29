// ARIS Interactive Canvas, Gallery & Chat Islands Runtime
(function () {
  "use strict";

  // State Management
  const state = {
    activeImage: null,
    activeMask: null,
    brushSize: 24,
    brushMode: "brush", // "brush" | "eraser"
    aspectRatio: "1:1",
    backend: "pollinations",
    model: "flux",
    steps: 30,
    cfgScale: 7.5,
    seed: -1,
    enableCritic: true,
    isDrawing: false,
    history: [],
    subagents: [],
  };

  // Initialize
  document.addEventListener("DOMContentLoaded", () => {
    loadSettings();
    initSSE();
    initCanvas();
    initGallery();
    initChat();
    initControls();
    fetchInitialData();
  });

  // Settings & LocalStorage
  function loadSettings() {
    try {
      const saved = localStorage.getItem("aris_settings");
      if (saved) {
        Object.assign(state, JSON.parse(saved));
      }
    } catch (e) {}
  }

  function saveSettings() {
    try {
      localStorage.setItem("aris_settings", JSON.stringify({
        aspectRatio: state.aspectRatio,
        backend: state.backend,
        model: state.model,
        steps: state.steps,
        cfgScale: state.cfgScale,
        seed: state.seed,
        enableCritic: state.enableCritic,
      }));
    } catch (e) {}
  }

  // Real-time SSE Connection
  function initSSE() {
    const urlParams = new URLSearchParams(window.location.search);
    const token = urlParams.get("token") || "";
    const sseUrl = token ? `/api/events?token=${encodeURIComponent(token)}` : "/api/events";

    let evtSource = new EventSource(sseUrl);

    evtSource.addEventListener("progress", (e) => {
      try {
        const data = JSON.parse(e.data);
        updateProgress(data);
      } catch (err) {}
    });

    evtSource.addEventListener("reasoning", (e) => {
      try {
        const data = JSON.parse(e.data);
        appendReasoning(data);
      } catch (err) {}
    });

    evtSource.addEventListener("image_ready", (e) => {
      try {
        const data = JSON.parse(e.data);
        handleImageReady(data);
      } catch (err) {}
    });

    evtSource.addEventListener("critic_evaluation", (e) => {
      try {
        const data = JSON.parse(e.data);
        handleCritic(data);
      } catch (err) {}
    });

    evtSource.addEventListener("error", (e) => {
      console.warn("SSE connection interrupted, retrying...");
    });
  }

  function updateProgress(data) {
    const statusText = document.getElementById("status-text");
    const progressBar = document.getElementById("progress-bar-fill");
    if (statusText && data.message) {
      statusText.innerText = data.message;
    }
    if (progressBar && data.percent !== undefined) {
      progressBar.style.width = `${data.percent}%`;
    }
  }

  function appendReasoning(data) {
    const chatFeed = document.getElementById("chat-feed");
    if (!chatFeed) return;

    let thoughtBlock = document.getElementById(`thought-${data.job_id}`);
    if (!thoughtBlock) {
      thoughtBlock = document.createElement("div");
      thoughtBlock.id = `thought-${data.job_id}`;
      thoughtBlock.className = "thought-accordion";
      thoughtBlock.innerHTML = `<strong>@${data.subagent || "director"} Reasoning:</strong><br><span class="thought-content"></span>`;
      chatFeed.appendChild(thoughtBlock);
      chatFeed.scrollTop = chatFeed.scrollHeight;
    }
    const content = thoughtBlock.querySelector(".thought-content");
    if (content && data.chunk) {
      content.innerText += " " + data.chunk;
      chatFeed.scrollTop = chatFeed.scrollHeight;
    }
  }

  function handleImageReady(data) {
    const chatFeed = document.getElementById("chat-feed");
    if (chatFeed) {
      const bubble = document.createElement("div");
      bubble.className = "chat-bubble aris";
      bubble.innerHTML = `<div>✨ <strong>Image Generated</strong> (${data.aspect_ratio || "1:1"})</div>
        <img src="${data.url}" class="mt-2 rounded max-h-48 cursor-pointer hover:opacity-90" onclick="window.arisViewImage('${data.url}', '${encodeURIComponent(data.prompt || "")}')" style="max-height: 180px; border-radius: 6px; margin-top: 6px;"/>
        <div class="text-xs text-muted mt-1 font-mono">${data.prompt || ""}</div>`;
      chatFeed.appendChild(bubble);
      chatFeed.scrollTop = chatFeed.scrollHeight;
    }

    // Load into center canvas if desired
    loadImageToCanvas(data.url);
    fetchHistory();
  }

  function handleCritic(data) {
    const chatFeed = document.getElementById("chat-feed");
    if (chatFeed) {
      const bubble = document.createElement("div");
      bubble.className = "chat-bubble aris";
      const badgeColor = data.passed ? "text-green-400" : "text-amber-400";
      bubble.innerHTML = `<div>👁️ <strong>Critic Score:</strong> <span class="${badgeColor} font-bold">${(data.score * 10).toFixed(1)}/10</span></div>
        <div class="text-xs mt-1 text-slate-300">${data.critique || ""}</div>`;
      chatFeed.appendChild(bubble);
      chatFeed.scrollTop = chatFeed.scrollHeight;
    }
  }

  // Interactive Center Canvas Island
  let imgCanvas, maskCanvas, offscreenMask;
  let imgCtx, maskCtx, offscreenCtx;

  function initCanvas() {
    imgCanvas = document.getElementById("image-canvas");
    maskCanvas = document.getElementById("mask-canvas");
    const dropzone = document.getElementById("canvas-dropzone");
    const fileInput = document.getElementById("canvas-file-input");

    if (!imgCanvas || !maskCanvas) return;

    imgCtx = imgCanvas.getContext("2d");
    maskCtx = maskCanvas.getContext("2d");

    offscreenMask = document.createElement("canvas");
    offscreenCtx = offscreenMask.getContext("2d");

    // Dropzone handlers
    if (dropzone) {
      dropzone.addEventListener("click", () => fileInput && fileInput.click());
      dropzone.addEventListener("dragover", (e) => {
        e.preventDefault();
        dropzone.classList.add("dragover");
      });
      dropzone.addEventListener("dragleave", () => dropzone.classList.remove("dragover"));
      dropzone.addEventListener("drop", (e) => {
        e.preventDefault();
        dropzone.classList.remove("dragover");
        if (e.dataTransfer.files && e.dataTransfer.files[0]) {
          loadFile(e.dataTransfer.files[0]);
        }
      });
    }

    if (fileInput) {
      fileInput.addEventListener("change", (e) => {
        if (e.target.files && e.target.files[0]) {
          loadFile(e.target.files[0]);
        }
      });
    }

    // Drawing handlers
    maskCanvas.addEventListener("mousedown", startDrawing);
    maskCanvas.addEventListener("mousemove", draw);
    window.addEventListener("mouseup", stopDrawing);

    // Touch events
    maskCanvas.addEventListener("touchstart", (e) => {
      const touch = e.touches[0];
      const mouseEvent = new MouseEvent("mousedown", {
        clientX: touch.clientX,
        clientY: touch.clientY,
      });
      maskCanvas.dispatchEvent(mouseEvent);
    });
    maskCanvas.addEventListener("touchmove", (e) => {
      const touch = e.touches[0];
      const mouseEvent = new MouseEvent("mousemove", {
        clientX: touch.clientX,
        clientY: touch.clientY,
      });
      maskCanvas.dispatchEvent(mouseEvent);
    });
    window.addEventListener("touchend", stopDrawing);
  }

  function loadFile(file) {
    const reader = new FileReader();
    reader.onload = (e) => {
      loadImageToCanvas(e.target.result);
    };
    reader.readAsDataURL(file);
  }

  function loadImageToCanvas(src) {
    const img = new Image();
    img.crossOrigin = "anonymous";
    img.onload = () => {
      state.activeImage = img;

      // Hide dropzone, show canvas
      const dropzone = document.getElementById("canvas-dropzone");
      const container = document.getElementById("canvas-container");
      if (dropzone) dropzone.style.display = "none";
      if (container) container.style.display = "inline-block";

      const maxW = 640;
      const maxH = 480;
      let w = img.width;
      let h = img.height;

      const scale = Math.min(maxW / w, maxH / h, 1);
      w = Math.round(w * scale);
      h = Math.round(h * scale);

      imgCanvas.width = w;
      imgCanvas.height = h;
      maskCanvas.width = w;
      maskCanvas.height = h;
      offscreenMask.width = w;
      offscreenMask.height = h;

      imgCtx.drawImage(img, 0, 0, w, h);
      clearMask();
    };
    img.src = src;
  }

  function startDrawing(e) {
    state.isDrawing = true;
    draw(e);
  }

  function draw(e) {
    if (!state.isDrawing || !maskCanvas) return;
    const rect = maskCanvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    maskCtx.lineWidth = state.brushSize;
    maskCtx.lineCap = "round";
    maskCtx.lineJoin = "round";

    offscreenCtx.lineWidth = state.brushSize;
    offscreenCtx.lineCap = "round";
    offscreenCtx.lineJoin = "round";

    if (state.brushMode === "eraser") {
      maskCtx.globalCompositeOperation = "destination-out";
      maskCtx.beginPath();
      maskCtx.arc(x, y, state.brushSize / 2, 0, Math.PI * 2);
      maskCtx.fill();

      offscreenCtx.globalCompositeOperation = "destination-out";
      offscreenCtx.beginPath();
      offscreenCtx.arc(x, y, state.brushSize / 2, 0, Math.PI * 2);
      offscreenCtx.fill();
    } else {
      maskCtx.globalCompositeOperation = "source-over";
      maskCtx.strokeStyle = "rgba(255, 0, 127, 0.5)"; // Magenta 50%
      maskCtx.beginPath();
      maskCtx.arc(x, y, state.brushSize / 2, 0, Math.PI * 2);
      maskCtx.fillStyle = "rgba(255, 0, 127, 0.5)";
      maskCtx.fill();

      offscreenCtx.globalCompositeOperation = "source-over";
      offscreenCtx.fillStyle = "#ffffff";
      offscreenCtx.beginPath();
      offscreenCtx.arc(x, y, state.brushSize / 2, 0, Math.PI * 2);
      offscreenCtx.fill();
    }
  }

  function stopDrawing() {
    state.isDrawing = false;
  }

  function clearMask() {
    if (maskCtx && offscreenCtx) {
      maskCtx.clearRect(0, 0, maskCanvas.width, maskCanvas.height);
      offscreenCtx.fillStyle = "#000000";
      offscreenCtx.fillRect(0, 0, offscreenMask.width, offscreenMask.height);
    }
  }

  // Left Panel - Gallery & Lightbox
  function initGallery() {
    const modal = document.getElementById("lightbox-modal");
    if (modal) {
      modal.addEventListener("click", (e) => {
        if (e.target === modal || e.target.classList.contains("close-modal")) {
          modal.classList.remove("active");
        }
      });
    }
  }

  window.arisViewImage = function (url, prompt) {
    const modal = document.getElementById("lightbox-modal");
    const img = document.getElementById("lightbox-img");
    const promptEl = document.getElementById("lightbox-prompt");
    if (modal && img) {
      img.src = url;
      if (promptEl) promptEl.innerText = decodeURIComponent(prompt);
      modal.classList.add("active");
    }
  };

  window.arisUseAsReference = function () {
    const img = document.getElementById("lightbox-img");
    if (img && img.src) {
      loadImageToCanvas(img.src);
      const modal = document.getElementById("lightbox-modal");
      if (modal) modal.classList.remove("active");
    }
  };

  function fetchHistory() {
    fetch("/api/history?limit=30")
      .then((res) => res.json())
      .then((data) => {
        state.history = data;
        renderGallery(data);
      })
      .catch(() => {});
  }

  function renderGallery(items) {
    const grid = document.getElementById("gallery-grid");
    if (!grid) return;
    grid.innerHTML = "";

    items.forEach((item) => {
      const card = document.createElement("div");
      card.className = "gallery-card";
      card.onclick = () => window.arisViewImage(item.image_url, encodeURIComponent(item.prompt));
      card.innerHTML = `
        <img src="${item.image_url}" loading="lazy" />
        <div class="gallery-card-badge">${item.aspect_ratio || "1:1"}</div>
      `;
      grid.appendChild(card);
    });
  }

  // Right Panel - Chat & Input Handlers
  function initChat() {
    const input = document.getElementById("chat-input");
    const sendBtn = document.getElementById("chat-send-btn");
    const inpaintBtn = document.getElementById("inpaint-btn");

    if (input) {
      input.addEventListener("keydown", (e) => {
        if (e.key === "Enter" && !e.shiftKey) {
          e.preventDefault();
          submitPrompt();
        }
      });
    }

    if (sendBtn) {
      sendBtn.addEventListener("click", submitPrompt);
    }

    if (inpaintBtn) {
      inpaintBtn.addEventListener("click", submitInpaint);
    }
  }

  function submitPrompt() {
    const input = document.getElementById("chat-input");
    if (!input) return;
    const prompt = input.value.trim();
    if (!prompt) return;

    // Append user message to chat feed
    const chatFeed = document.getElementById("chat-feed");
    if (chatFeed) {
      const bubble = document.createElement("div");
      bubble.className = "chat-bubble user";
      bubble.innerText = prompt;
      chatFeed.appendChild(bubble);
      chatFeed.scrollTop = chatFeed.scrollHeight;
    }

    input.value = "";

    const payload = {
      prompt: prompt,
      backend: state.backend,
      model: state.model,
      aspect_ratio: state.aspectRatio,
      steps: parseInt(state.steps, 10),
      cfg_scale: parseFloat(state.cfgScale),
      seed: parseInt(state.seed, 10),
      enable_critic: state.enableCritic,
    };

    fetch("/api/generate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    }).catch((err) => {
      console.error("Generate error:", err);
    });
  }

  function submitInpaint() {
    const input = document.getElementById("chat-input");
    const prompt = input ? input.value.trim() : "";
    if (!prompt) {
      alert("Please enter a prompt describing the inpainting area.");
      return;
    }
    if (!imgCanvas || !offscreenMask) {
      alert("Please load an image before inpainting.");
      return;
    }

    imgCanvas.toBlob((imgBlob) => {
      offscreenMask.toBlob((maskBlob) => {
        const formData = new FormData();
        formData.append("prompt", prompt);
        formData.append("backend", state.backend);
        formData.append("model", state.model);
        formData.append("image", imgBlob, "image.png");
        formData.append("mask", maskBlob, "mask.png");

        fetch("/api/inpaint", {
          method: "POST",
          body: formData,
        }).catch((err) => {
          console.error("Inpaint error:", err);
        });
      }, "image/png");
    }, "image/png");
  }

  // Controls Initialization
  function initControls() {
    const ratioBtns = document.querySelectorAll(".ratio-btn");
    ratioBtns.forEach((btn) => {
      btn.addEventListener("click", () => {
        ratioBtns.forEach((b) => b.classList.remove("btn-primary"));
        btn.classList.add("btn-primary");
        state.aspectRatio = btn.getAttribute("data-ratio") || "1:1";
        saveSettings();
      });
    });

    const brushSizeSlider = document.getElementById("brush-size");
    if (brushSizeSlider) {
      brushSizeSlider.addEventListener("input", (e) => {
        state.brushSize = parseInt(e.target.value, 10);
      });
    }

    const brushBtn = document.getElementById("btn-brush");
    const eraserBtn = document.getElementById("btn-eraser");
    const clearBtn = document.getElementById("btn-clear-mask");

    if (brushBtn && eraserBtn) {
      brushBtn.addEventListener("click", () => {
        state.brushMode = "brush";
        brushBtn.classList.add("btn-primary");
        eraserBtn.classList.remove("btn-primary");
      });
      eraserBtn.addEventListener("click", () => {
        state.brushMode = "eraser";
        eraserBtn.classList.add("btn-primary");
        brushBtn.classList.remove("btn-primary");
      });
    }

    if (clearBtn) {
      clearBtn.addEventListener("click", clearMask);
    }
  }

  function fetchInitialData() {
    fetchHistory();
    fetch("/api/subagents")
      .then((r) => r.json())
      .then((subs) => {
        state.subagents = subs;
        renderSubagentBadges(subs);
      })
      .catch(() => {});

    fetch("/api/backends")
      .then((r) => r.json())
      .then((backends) => {
        populateBackends(backends);
      })
      .catch(() => {});
  }

  function renderSubagentBadges(subs) {
    const container = document.getElementById("subagent-badges");
    if (!container) return;
    container.innerHTML = "";
    subs.forEach((s) => {
      const badge = document.createElement("span");
      badge.className = "subagent-badge";
      badge.innerText = `@${s.name}`;
      badge.title = s.description;
      badge.onclick = () => {
        const input = document.getElementById("chat-input");
        if (input) {
          input.value = `@${s.name} ` + input.value;
          input.focus();
        }
      };
      container.appendChild(badge);
    });
  }

  function populateBackends(backends) {
    const select = document.getElementById("backend-select");
    if (!select) return;
    select.innerHTML = "";
    backends.forEach((b) => {
      const opt = document.createElement("option");
      opt.value = b.name;
      opt.innerText = b.name + (b.is_default ? " (default)" : "");
      if (b.name === state.backend) opt.selected = true;
      select.appendChild(opt);
    });
  }
})();
