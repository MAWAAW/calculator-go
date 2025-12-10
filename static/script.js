window.calculate = async function(op) {
  const a = parseFloat(document.getElementById("a").value);
  const b = parseFloat(document.getElementById("b").value);

  try {
    const res = await fetch("/api/calc", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ a, b, op })
    });

    if (!res.ok) {
      const text = await res.text();
      alert("Erreur : " + text);
      return;
    }

    const data = await res.json();
    document.getElementById("result").textContent = data.result;
  } catch (err) {
    alert("Erreur réseau ou serveur : " + err);
  }
}
