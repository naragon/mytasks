// Initialize Kanban board and other sortables
document.addEventListener('DOMContentLoaded', function() {
    initializeKanban();
    initializeSidebarSortable();
    initializeSidebarControls();
    initializeFormTriggers();
});

// Re-initialize after htmx swaps
document.addEventListener('htmx:afterSwap', function() {
    initializeKanban();
    initializeSidebarControls();
    initializeFormTriggers();
});

const sidebarWidthStorageKey = 'mytasks.sidebar.width';
const sidebarCollapsedStorageKey = 'mytasks.sidebar.collapsed';
const sidebarMinWidth = 200;
const sidebarMaxWidth = 460;

function initializeSidebarControls() {
    const layout = document.querySelector('.app-layout');
    const sidebar = document.querySelector('.sidebar');
    if (!layout || !sidebar) return;

    applyStoredSidebarState(layout);
    bindSidebarToggle(layout);
    bindSidebarStepResize(layout);
    bindSidebarResizer(layout, sidebar);
    updateSidebarToggleUI(layout);
}

function bindSidebarToggle(layout) {
    document.querySelectorAll('[data-action="toggle-sidebar"]').forEach(function(button) {
        if (button.dataset.bound === '1') return;
        button.dataset.bound = '1';
        button.addEventListener('click', function(event) {
            event.preventDefault();
            setSidebarCollapsed(layout, !layout.classList.contains('sidebar-collapsed'));
        });
    });
}

function bindSidebarResizer(layout, sidebar) {
    const resizer = sidebar.querySelector('[data-action="resize-sidebar"]');
    if (!resizer || resizer.dataset.bound === '1') return;

    resizer.dataset.bound = '1';
    resizer.addEventListener('mousedown', function(event) {
        if (window.matchMedia('(max-width: 600px)').matches) return;

        event.preventDefault();
        setSidebarCollapsed(layout, false);
        document.body.classList.add('resizing-sidebar');

        const onMouseMove = function(moveEvent) {
            const sidebarLeft = sidebar.getBoundingClientRect().left;
            const newWidth = moveEvent.clientX - sidebarLeft;
            setSidebarWidth(newWidth);
        };

        const onMouseUp = function() {
            document.body.classList.remove('resizing-sidebar');
            document.removeEventListener('mousemove', onMouseMove);
            document.removeEventListener('mouseup', onMouseUp);
        };

        document.addEventListener('mousemove', onMouseMove);
        document.addEventListener('mouseup', onMouseUp);
    });
}

function bindSidebarStepResize(layout) {
    const step = 24;

    document.querySelectorAll('[data-action="widen-sidebar"], [data-action="narrow-sidebar"]').forEach(function(button) {
        if (button.dataset.bound === '1') return;
        button.dataset.bound = '1';
        button.addEventListener('click', function(event) {
            event.preventDefault();
            setSidebarCollapsed(layout, false);

            const direction = button.getAttribute('data-action') === 'widen-sidebar' ? 1 : -1;
            const currentWidth = getCurrentSidebarWidth();
            setSidebarWidth(currentWidth + (direction * step));
        });
    });
}

function applyStoredSidebarState(layout) {
    try {
        const savedWidth = parseInt(localStorage.getItem(sidebarWidthStorageKey), 10);
        if (!isNaN(savedWidth)) {
            setSidebarWidth(savedWidth, false);
        }

        const isCollapsed = localStorage.getItem(sidebarCollapsedStorageKey) === '1';
        layout.classList.toggle('sidebar-collapsed', isCollapsed);
    } catch (_error) {
        // Ignore storage failures.
    }
}

function setSidebarCollapsed(layout, collapsed) {
    layout.classList.toggle('sidebar-collapsed', collapsed);
    updateSidebarToggleUI(layout);

    try {
        localStorage.setItem(sidebarCollapsedStorageKey, collapsed ? '1' : '0');
    } catch (_error) {
        // Ignore storage failures.
    }
}

function setSidebarWidth(width, persist) {
    const clamped = Math.max(sidebarMinWidth, Math.min(sidebarMaxWidth, Math.round(width)));
    document.documentElement.style.setProperty('--sidebar-width', clamped + 'px');

    if (persist === false) return;

    try {
        localStorage.setItem(sidebarWidthStorageKey, String(clamped));
    } catch (_error) {
        // Ignore storage failures.
    }
}

function getCurrentSidebarWidth() {
    const fromVariable = parseInt(getComputedStyle(document.documentElement).getPropertyValue('--sidebar-width'), 10);
    if (!isNaN(fromVariable)) {
        return fromVariable;
    }

    const sidebar = document.querySelector('.sidebar');
    if (!sidebar) {
        return sidebarMinWidth;
    }

    return Math.round(sidebar.getBoundingClientRect().width);
}

function updateSidebarToggleUI(layout) {
    const collapsed = layout.classList.contains('sidebar-collapsed');

    document.querySelectorAll('[data-action="toggle-sidebar"]').forEach(function(button) {
        button.textContent = collapsed ? '›' : '‹';
        button.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
        button.setAttribute('aria-label', collapsed ? 'Expand navigation' : 'Collapse navigation');
        button.setAttribute('title', collapsed ? 'Expand navigation' : 'Collapse navigation');
    });
}

function initializeKanban() {
    const kanbanPage = document.querySelector('.kanban-page');
    if (!kanbanPage) return;

    const projectId = kanbanPage.dataset.projectId;

    document.querySelectorAll('.kanban-cards').forEach(function(column) {
        if (column._sortable) return;

        column._sortable = new Sortable(column, {
            group: 'kanban',
            animation: 150,
            ghostClass: 'sortable-ghost',
            dragClass: 'sortable-drag',
            chosenClass: 'sortable-chosen',
            onEnd: function(evt) {
                const taskId = parseInt(evt.item.dataset.id);
                const newStatus = evt.to.dataset.status;
                const oldStatus = evt.from.dataset.status;

                // If moved to a different column, update status
                if (oldStatus !== newStatus) {
                    const newIndex = Array.from(evt.to.querySelectorAll('.kanban-card'))
                        .findIndex(card => parseInt(card.dataset.id) === taskId);

                    fetch('/api/tasks/' + taskId + '/move', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                            status: newStatus,
                            sort_order: newIndex + 1
                        })
                    });

                    // Keep inline edit form state in sync with card column immediately.
                    syncKanbanCardEditStatus(evt.item, newStatus);
                }

                // Reorder within the target column
                const ids = Array.from(evt.to.querySelectorAll('.kanban-card'))
                    .map(card => parseInt(card.dataset.id));

                fetch('/api/projects/' + projectId + '/tasks/reorder?status=' + newStatus, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ ids: ids })
                });

                // If moved between columns, also reorder the source column
                if (oldStatus !== newStatus) {
                    const sourceIds = Array.from(evt.from.querySelectorAll('.kanban-card'))
                        .map(card => parseInt(card.dataset.id));

                    fetch('/api/projects/' + projectId + '/tasks/reorder?status=' + oldStatus, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ ids: sourceIds })
                    });
                }

                // Update column counts
                updateKanbanCounts();
            }
        });
    });
}

function syncKanbanCardEditStatus(card, status) {
    const statusSelect = card.querySelector('.kanban-card-edit select[name="status"]');
    if (statusSelect) {
        statusSelect.value = status;
    }
}

function updateKanbanCounts() {
    document.querySelectorAll('.kanban-column').forEach(function(col) {
        const count = col.querySelectorAll('.kanban-card').length;
        const badge = col.querySelector('.kanban-count');
        if (badge) badge.textContent = count;
    });
}

function initializeSidebarSortable() {
    const sidebarList = document.getElementById('sidebar-projects');
    if (sidebarList && !sidebarList._sortable) {
        sidebarList._sortable = new Sortable(sidebarList, {
            animation: 150,
            ghostClass: 'sortable-ghost',
            onEnd: function() {
                const ids = Array.from(sidebarList.querySelectorAll('.sidebar-item'))
                    .map(function(item) {
                        var link = item.querySelector('a');
                        var href = link ? link.getAttribute('href') : '';
                        var match = href.match(/\/projects\/(\d+)/);
                        return match ? parseInt(match[1]) : 0;
                    })
                    .filter(function(id) { return id > 0; });

                fetch('/api/projects/reorder', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ ids: ids })
                });
            }
        });
    }
}

function initializeFormTriggers() {
    document.querySelectorAll('[data-action="show-project-form"], button[onclick*="showProjectForm"]').forEach(function(button) {
        if (button.dataset.bound === '1') return;
        button.dataset.bound = '1';
        // Backward-compat: strip legacy inline handlers that call showProjectForm().
        button.removeAttribute('onclick');
        button.addEventListener('click', function(event) {
            event.preventDefault();
            showProjectForm();
        });
    });
}

// Form visibility functions
function showProjectForm() {
    var form = document.getElementById('new-project-form');
    if (form) {
        form.classList.remove('hidden');
        var input = form.querySelector('input[name="name"]');
        if (input) input.focus();
    }
}

function showEditProjectForm() {
    var form = document.getElementById('edit-project-form');
    if (form) {
        form.classList.remove('hidden');
        var input = form.querySelector('input[name="name"]');
        if (input) input.focus();
    }
}

function showKanbanTaskForm(status) {
    // Hide all kanban forms first
    document.querySelectorAll('[id^="kanban-form-"]').forEach(function(f) {
        f.classList.add('hidden');
    });

    var form = document.getElementById('kanban-form-' + status);
    if (form) {
        form.classList.remove('hidden');
        var input = form.querySelector('input[name="description"]');
        if (input) input.focus();
    }
}

function toggleKanbanCardEdit(taskId) {
    var editForm = document.getElementById('kanban-card-edit-' + taskId);
    if (!editForm) return;

    var isHidden = editForm.classList.contains('hidden');

    // Close all other open card edits
    document.querySelectorAll('.kanban-card-edit').forEach(function(f) {
        f.classList.add('hidden');
    });

    if (isHidden) {
        editForm.classList.remove('hidden');
        var input = editForm.querySelector('input[name="description"]');
        if (input) {
            input.focus();
            input.select();
        }
    }
}

function hideForm(button) {
    var formContainer = button.closest('.form-container');
    if (formContainer) {
        formContainer.classList.add('hidden');
        var form = formContainer.querySelector('form');
        if (form) form.reset();
    }
}

// Expose handlers for inline onclick attributes used by templates.
window.showProjectForm = showProjectForm;
window.showEditProjectForm = showEditProjectForm;
window.showKanbanTaskForm = showKanbanTaskForm;
window.toggleKanbanCardEdit = toggleKanbanCardEdit;
window.hideForm = hideForm;

// Handle HX-Redirect responses
document.addEventListener('htmx:beforeSwap', function(event) {
    var redirect = event.detail.xhr.getResponseHeader('HX-Redirect');
    if (redirect) {
        window.location.href = redirect;
    }
});
