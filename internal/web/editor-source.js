import { EditorView, basicSetup } from "codemirror"
import { EditorState } from "@codemirror/state"

// --- Preview Logic ---

// Debounce function to limit how often the preview is updated
let debounceTimer;
function debounce(func, timeout = 300){
  return (...args) => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => { func.apply(this, args); }, timeout);
  };
}

// Function to fetch and render the preview
async function updatePreview(docText) {
    const previewPane = document.querySelector("#preview-content");
    if (!previewPane) return;

    try {
        const response = await fetch('/_preview', {
            method: 'POST',
            headers: {
                'Content-Type': 'text/plain; charset=utf-8',
            },
            body: docText
        });
        if (response.ok) {
            const html = await response.text();
            previewPane.innerHTML = html;
        } else {
            previewPane.innerHTML = '<div class="alert alert-danger">Error loading preview.</div>';
        }
    } catch (error) {
        console.error('Preview Error:', error);
        previewPane.innerHTML = '<div class="alert alert-danger">Could not connect to server for preview.</div>';
    }
}

const debouncedUpdatePreview = debounce(updatePreview, 250);

// --- Editor Setup ---

const textarea = document.querySelector("#content");
const form = document.querySelector("#editForm");
const finalSaveButton = document.querySelector("#finalSaveButton");

const state = EditorState.create({
    doc: textarea.value,
    extensions: [
        basicSetup,
        EditorView.theme({
            "&": {
                color: "var(--bs-body-color)",
                backgroundColor: "var(--bs-body-bg)",
                height: "100%",
            },
            ".cm-scroller": {
                fontFamily: "'JetBrains Mono', monospace",
                overflow: "auto"
            },
            ".cm-content": {
                caretColor: "var(--bs-body-color)"
            },
            "&.cm-focused": {
                outline: "0",
            },
            ".cm-gutters": {
                backgroundColor: "transparent",
                color: "var(--bs-secondary-color)",
                border: "none"
            },
            ".cm-cursor, .cm-dropCursor": {
                borderLeftColor: "var(--bs-body-color)"
            },
            ".cm-activeLineGutter": {
                backgroundColor: "transparent"
            },
            ".cm-activeLine": {
                backgroundColor: "transparent"
            },
            "& .cm-selectionBackground, ::selection": {
                backgroundColor: "rgba(var(--bs-primary-rgb), 0.2) !important",
            },
        }),
        // Add a listener that calls the preview function on document changes
        EditorView.updateListener.of(update => {
            if (update.docChanged) {
                debouncedUpdatePreview(update.state.doc.toString());
            }
        })
    ]
});

const editorParent = textarea.parentElement;
const view = new EditorView({
    state,
    parent: editorParent
});

// --- Form Submission Logic ---

if (finalSaveButton && form) {
    finalSaveButton.addEventListener('click', () => {
        textarea.value = view.state.doc.toString();
        const comment = document.querySelector("#modalComment").value;
        let commentInput = form.querySelector('input[name="comment"]');
        if (!commentInput) {
            commentInput = document.createElement('input');
            commentInput.type = 'hidden';
            commentInput.name = 'comment';
            form.appendChild(commentInput);
        }
        commentInput.value = comment;
        form.submit();
    });
}

// --- Initial Load ---

// Trigger an initial preview render when the page loads
updatePreview(textarea.value);
