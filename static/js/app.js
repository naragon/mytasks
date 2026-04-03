// Initialize Kanban board and other sortables
document.addEventListener('DOMContentLoaded', function() {
    initializeKanban();
    initializeSidebarSortable();
});

// Re-initialize after htmx swaps
document.addEventListener('htmx:afterSwap', function() {
    initializeKanban();
});

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

// Handle HX-Redirect responses
document.addEventListener('htmx:beforeSwap', function(event) {
    var redirect = event.detail.xhr.getResponseHeader('HX-Redirect');
    if (redirect) {
        window.location.href = redirect;
    }
});
