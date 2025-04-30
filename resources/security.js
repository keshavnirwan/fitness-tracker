async function loadChatHistory(receiver) {
    const response = await fetch(`/chat-history?receiver=${receiver}`);
    const history = await response.json();
  
    const chatBox = document.getElementById("chatBox");
    chatBox.innerHTML = "";
    history.forEach(msg => {
      const p = document.createElement("p");
      p.textContent = `${msg.Time} - ${msg.Sender}: ${msg.Content}`;
      chatBox.appendChild(p);
    });
  }
   