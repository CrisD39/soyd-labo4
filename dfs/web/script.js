// Carga la lista de archivos desde /api/ls y actualiza la tabla
async function loadFiles() {
  const status = document.getElementById('status');
  const tbody = document.getElementById('files-body');

  status.textContent = 'Cargando...';

  try {
    const res = await fetch('/api/ls');
    if (!res.ok) throw new Error('HTTP ' + res.status);

    const data = await res.json();
    const files = Array.isArray(data.files) ? data.files : [];

    tbody.innerHTML = '';

    if (files.length === 0) {
      const tr = document.createElement('tr');
      const td = document.createElement('td');
      td.colSpan = 2;
      td.textContent = '(No hay archivos en el DFS)';
      tr.appendChild(td);
      tbody.appendChild(tr);
    } else {
      for (const name of files) {
        const tr = document.createElement('tr');

        const tdName = document.createElement('td');
        tdName.textContent = name;

        const tdActions = document.createElement('td');
        const a = document.createElement('a');
        a.href = '/files/download?name=' + encodeURIComponent(name);
        a.textContent = 'Descargar';
        a.className = 'download-link';

        tdActions.appendChild(a);

        tr.appendChild(tdName);
        tr.appendChild(tdActions);
        tbody.appendChild(tr);
      }
    }

    status.textContent = `Mostrando ${files.length} archivo(s).`;
  } catch (err) {
    console.error(err);
    status.textContent = 'Error cargando lista: ' + err.message;
  }
}

// Todo el resto se engancha cuando el DOM está listo
document.addEventListener('DOMContentLoaded', () => {
  const refreshBtn = document.getElementById('refresh');
  const dropzone = document.getElementById('dropzone');
  const fileInput = document.getElementById('file-input');
  const remoteNameInput = document.getElementById('remote-name');
  const uploadInfo = document.getElementById('upload-file-info');
  const uploadForm = document.getElementById('upload-form');

  let selectedFile = null;

  // Botón de refrescar lista
  if (refreshBtn) {
    refreshBtn.addEventListener('click', loadFiles);
  }

  // ---------- Lógica de upload con dropzone ----------

  if (dropzone && fileInput) {
    // Click en dropzone -> abrir selector de archivo
    dropzone.addEventListener('click', () => {
      fileInput.click();
    });

    // Manejar selección desde el input file
    fileInput.addEventListener('change', () => {
      if (fileInput.files && fileInput.files.length > 0) {
        selectedFile = fileInput.files[0];
        uploadInfo.textContent =
          `Archivo seleccionado: ${selectedFile.name} (${selectedFile.size} bytes)`;
        // Si no puso nombre remoto, usamos el nombre del archivo
        if (!remoteNameInput.value) {
          remoteNameInput.value = selectedFile.name;
        }
      }
    });

    // Drag & drop
    ['dragenter', 'dragover'].forEach(evName => {
      dropzone.addEventListener(evName, (e) => {
        e.preventDefault();
        e.stopPropagation();
        dropzone.classList.add('dragover');
      });
    });

    ['dragleave', 'dragend', 'drop'].forEach(evName => {
      dropzone.addEventListener(evName, (e) => {
        e.preventDefault();
        e.stopPropagation();
        dropzone.classList.remove('dragover');
      });
    });

    dropzone.addEventListener('drop', (e) => {
      const files = e.dataTransfer.files;
      if (files && files.length > 0) {
        selectedFile = files[0];
        uploadInfo.textContent =
          `Archivo seleccionado: ${selectedFile.name} (${selectedFile.size} bytes)`;
        if (!remoteNameInput.value) {
          remoteNameInput.value = selectedFile.name;
        }
      }
    });
  }

  // Submit del formulario -> POST /files/upload
  if (uploadForm) {
    uploadForm.addEventListener('submit', async (e) => {
      e.preventDefault();

      if (!selectedFile) {
        alert('Primero elegí un archivo (drop o click).');
        return;
      }
      if (!remoteNameInput.value.trim()) {
        alert('Ingresá el nombre con el que querés subir el archivo.');
        return;
      }

      const status = document.getElementById('status');
      status.textContent = 'Subiendo archivo...';

      try {
        const formData = new FormData();
        formData.append('remote', remoteNameInput.value.trim());
        formData.append('file', selectedFile);

        const res = await fetch('/files/upload', {
          method: 'POST',
          body: formData,
        });

        if (!res.ok) {
          const text = await res.text();
          throw new Error(`HTTP ${res.status}: ${text}`);
        }

        const data = await res.json();
        status.textContent = `Archivo subido como "${data.remote}".`;
        // Limpiamos selección
        selectedFile = null;
        uploadInfo.textContent = '';
        fileInput.value = '';
        // Opcional: no limpiar remoteName para poder subir versiones
        // remoteNameInput.value = '';

        // Refrescamos la lista de archivos
        loadFiles();
      } catch (err) {
        console.error(err);
        status.textContent = 'Error subiendo archivo: ' + err.message;
        alert('Error subiendo archivo: ' + err.message);
      }
    });
  }

  // Carga inicial de la lista
  loadFiles();
});
