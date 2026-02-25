// Initialize SortableJS for drag-and-drop reordering

document.addEventListener('DOMContentLoaded', function() {
    initializeSortable();
});

// Re-initialize after htmx swaps
document.addEventListener('htmx:afterSwap', function() {
    initializeSortable();
});

function initializeSortable() {
    // Projects list on home page
    const projectsList = document.getElementById('projects-list');
    if (projectsList && !projectsList.sortable) {
        projectsList.sortable = new Sortable(projectsList, {
            handle: '.drag-handle',
            animation: 150,
            ghostClass: 'sortable-ghost',
            dragClass: 'sortable-drag',
            onEnd: function(evt) {
                const ids = Array.from(projectsList.querySelectorAll('.project-card'))
                    .map(card => parseInt(card.dataset.id));

                fetch('/api/projects/reorder', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ ids: ids }),
                });
            }
        });
    }

    // Tasks list on project detail page
    const tasksList = document.getElementById('tasks-list');
    if (tasksList && !tasksList.sortable) {
        const projectId = tasksList.dataset.projectId;
        tasksList.sortable = new Sortable(tasksList, {
            handle: '.drag-handle',
            animation: 150,
            ghostClass: 'sortable-ghost',
            dragClass: 'sortable-drag',
            onEnd: function(evt) {
                const ids = Array.from(tasksList.querySelectorAll('.task-item'))
                    .map(task => parseInt(task.dataset.id));

                fetch(`/api/projects/${projectId}/tasks/reorder`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ ids: ids }),
                });
            }
        });
    }
}

// Form visibility functions
function showProjectForm() {
    const form = document.getElementById('new-project-form');
    if (form) {
        form.classList.remove('hidden');
        form.querySelector('input[name="name"]').focus();
    }
}

function showEditProjectForm(projectId) {
    const form = document.getElementById('edit-project-form');
    if (form) {
        form.classList.remove('hidden');
        form.querySelector('input[name="name"]').focus();
    }
}

function showTaskForm(projectId) {
    const form = document.getElementById('new-task-form');
    if (form) {
        form.classList.remove('hidden');
        form.querySelector('input[name="description"]').focus();
    }
}

function showUpcomingTaskForm() {
    const form = document.getElementById('new-upcoming-task-form');
    if (form) {
        form.classList.remove('hidden');
        form.querySelector('input[name="description"]').focus();
    }
}

function showEditTaskForm(taskId, projectId) {
    const taskElement = document.getElementById(`task-${taskId}`);
    if (!taskElement) {
        return;
    }

    const formContainer = taskElement.querySelector(`#edit-task-form-${taskId}`);
    if (!formContainer) {
        return;
    }

    const shouldOpen = formContainer.classList.contains('hidden');

    document.querySelectorAll('.inline-task-form').forEach(function(form) {
        if (form !== formContainer) {
            form.classList.add('hidden');
        }
    });

    if (shouldOpen) {
        formContainer.classList.remove('hidden');
    } else {
        formContainer.classList.add('hidden');
    }

    if (!formContainer.classList.contains('hidden')) {
        const descriptionInput = formContainer.querySelector('input[name="description"]');
        if (descriptionInput) {
            descriptionInput.focus();
            descriptionInput.select();
        }
    }
}

function toggleInlineTaskEdit(taskId) {
    const formContainer = document.getElementById(`inline-task-edit-${taskId}`);
    if (!formContainer) {
        return;
    }

    const shouldOpen = formContainer.classList.contains('hidden');

    document.querySelectorAll('.inline-edit-form').forEach(function(form) {
        if (form !== formContainer) {
            form.classList.add('hidden');
        }
    });

    if (shouldOpen) {
        formContainer.classList.remove('hidden');
        const input = formContainer.querySelector('input[name="description"]');
        if (input) {
            input.focus();
            input.select();
        }
    } else {
        formContainer.classList.add('hidden');
    }
}

function hideForm(button) {
    const formContainer = button.closest('.form-container');
    if (formContainer) {
        formContainer.classList.add('hidden');
        const form = formContainer.querySelector('form');
        if (form) {
            form.reset();
        }
    }
}

// Toggle target date visibility based on project type
function toggleTargetDate(select) {
    const targetDateGroup = document.getElementById('target-date-group');
    const targetDateInput = document.getElementById('project-target-date');

    if (select.value === 'category') {
        if (targetDateGroup) {
            targetDateGroup.style.opacity = '0.5';
        }
        if (targetDateInput) {
            targetDateInput.value = '';
            targetDateInput.disabled = true;
        }
    } else {
        if (targetDateGroup) {
            targetDateGroup.style.opacity = '1';
        }
        if (targetDateInput) {
            targetDateInput.disabled = false;
        }
    }
}

function clearCompletedFilters() {
    const startDateInput = document.getElementById('start-date');
    const endDateInput = document.getElementById('end-date');

    if (startDateInput) {
        startDateInput.value = '';
    }
    if (endDateInput) {
        endDateInput.value = '';
    }

    window.location.href = '/?tab=completed';
}

// Handle delete project redirect
document.addEventListener('htmx:beforeSwap', function(event) {
    if (event.detail.xhr.getResponseHeader('HX-Redirect')) {
        window.location.href = event.detail.xhr.getResponseHeader('HX-Redirect');
    }
});
