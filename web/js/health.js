document.addEventListener('DOMContentLoaded', () => {
    const loadingEl = document.getElementById('health-loading');
    const errorEl = document.getElementById('health-error');
    const tableEl = document.getElementById('health-table');
    const tbody = tableEl.querySelector('tbody');

    function addRow(name, ok) {
        const tr = document.createElement('tr');
        const statusClass = ok ? 'status-ok' : 'status-fail';
        const statusText = ok ? 'OK' : 'FAIL';
        tr.innerHTML = `<td>${name}</td><td class="${statusClass}">${statusText}</td>`;
        tbody.appendChild(tr);
    }

    fetch('/api/health')
        .then(resp => {
            if (!resp.ok) throw new Error('Failed to load health information');
            return resp.json();
        })
        .then(data => {
            loadingEl.style.display = 'none';
            tableEl.style.display = 'table';
            addRow('Server', data.server === 'ok');
            addRow('Database', data.database === true);
            if (data.feeds) {
                Object.entries(data.feeds).forEach(([feed, ok]) => {
                    addRow(`Feed: ${feed}`, ok);
                });
            }
        })
        .catch(err => {
            loadingEl.style.display = 'none';
            errorEl.textContent = err.message;
            errorEl.style.display = 'block';
        });
});
