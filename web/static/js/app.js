// LogStream Web UI JavaScript

// Configuration
const config = {
    apiBaseUrl: '/api/v1',
    refreshInterval: 30000, // 30 seconds
    maxLogsPerPage: 50,
    autoRefreshEnabled: false,
    refreshTimer: null,
    currentPage: 1,
    totalPages: 1,
    chartColors: {
        blue: 'rgba(54, 162, 235, 0.5)',
        red: 'rgba(255, 99, 132, 0.5)',
        green: 'rgba(75, 192, 192, 0.5)',
        yellow: 'rgba(255, 205, 86, 0.5)',
        purple: 'rgba(153, 102, 255, 0.5)',
        orange: 'rgba(255, 159, 64, 0.5)',
        grey: 'rgba(201, 203, 207, 0.5)'
    },
    volumeChart: null,
    levelsChart: null
};

// DOM Elements - Initialize after DOM is loaded
let elements = {};

// Initialize the application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    initApp();
});

// Main initialization function
function initApp() {
    // Initialize DOM elements
    elements = {
        logsTable: document.getElementById('logs-body'),
        logsPagination: document.getElementById('logs-pagination'),
        searchInput: document.getElementById('search-input'),
        searchButton: document.getElementById('search-button'),
        refreshButton: document.getElementById('refresh-logs'),
        autoRefreshButton: document.getElementById('auto-refresh'),
        exportButton: document.getElementById('export-logs'),
        sourcesList: document.querySelector('.list-sources'),
        timeRangeSelect: document.getElementById('time-range'),
        customRangeDiv: document.getElementById('custom-range'),
        startTimeInput: document.getElementById('start-time'),
        endTimeInput: document.getElementById('end-time'),
        levelCheckboxes: document.querySelectorAll('input[id^="level-"]'),
        volumeChartCanvas: document.getElementById('logs-volume-chart'),
        levelsChartCanvas: document.getElementById('logs-levels-chart'),
        
        // Modal elements
        logDetailModal: document.getElementById('logDetailModal') ? new bootstrap.Modal(document.getElementById('logDetailModal')) : null,
        detailTimestamp: document.getElementById('detail-timestamp'),
        detailLevel: document.getElementById('detail-level'),
        detailSource: document.getElementById('detail-source'),
        detailMessage: document.getElementById('detail-message'),
        detailRaw: document.getElementById('detail-raw'),
        detailFields: document.getElementById('detail-fields')
    };
    
    console.log("LogStream Web UI initializing...");
    
    // Load initial data
    loadSources();
    loadLogs();
    
    // Initialize charts
    initCharts();
    
    // Set up event listeners
    setupEventListeners();
    
    console.log("LogStream Web UI initialized successfully");
}

// Load log sources from API
function loadSources() {
    fetch(`${config.apiBaseUrl}/logs/sources`)
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error! Status: ${response.status}`);
            }
            return response.json();
        })
        .then(sources => {
            renderSources(sources);
        })
        .catch(error => {
            console.error('Error loading sources:', error);
            showErrorMessage('Failed to load log sources. Please try refreshing the page.');
        });
}

// Render sources list
function renderSources(sources) {
    if (!sources || sources.length === 0) {
        elements.sourcesList.innerHTML = '<li class="list-group-item">No sources available</li>';
        return;
    }
    
    let html = '';
    sources.forEach(source => {
        html += `
            <li class="list-group-item d-flex justify-content-between align-items-center source-item" data-source="${source}">
                <span class="source-name">${source}</span>
                <span class="badge bg-primary rounded-pill source-count">0</span>
            </li>
        `;
    });
    
    elements.sourcesList.innerHTML = html;
    
    // Add click event to source items
    document.querySelectorAll('.source-item').forEach(item => {
        item.addEventListener('click', function() {
            // Toggle active class
            this.classList.toggle('active');
            // Reload logs with the selected sources
            loadLogs();
        });
    });
}

// Load logs from API
function loadLogs() {
    showLoading();
    
    // Build query parameters
    const params = buildQueryParams();
    
    console.log('Fetching logs with params:', params);
    
    fetch(`${config.apiBaseUrl}/logs?${params}`)
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error! Status: ${response.status}`);
            }
            return response.json();
        })
        .then(logs => {
            console.log('Received logs:', logs ? logs.length : 0, 'entries');
            renderLogs(logs || []);
            updateSourceCounts(logs || []);
            updateCharts(logs || []);
            hideLoading();
        })
        .catch(error => {
            console.error('Error loading logs:', error);
            showErrorMessage('Failed to load logs. Please try refreshing the page.');
            hideLoading();
        });
}

// Build query parameters based on current filters
function buildQueryParams() {
    const params = new URLSearchParams();
    
    // Add search query if any
    const searchQuery = elements.searchInput ? elements.searchInput.value.trim() : "";
    if (searchQuery) {
        params.append('filter', searchQuery);
    }
    
    // Add selected sources
    const selectedSources = [];
    const sourceItems = document.querySelectorAll('.source-item.active');
    console.log(`Selected sources: ${sourceItems.length} items`);
    
    sourceItems.forEach(item => {
        const source = item.dataset.source;
        if (source) {
            selectedSources.push(source);
            console.log(`Adding source to filter: ${source}`);
        }
    });
    
    if (selectedSources.length > 0) {
        params.append('sources', selectedSources.join(','));
    }
    
    // Add selected levels
    const selectedLevels = [];
    if (elements.levelCheckboxes) {
        elements.levelCheckboxes.forEach(checkbox => {
            if (checkbox.checked) {
                selectedLevels.push(checkbox.value);
                console.log(`Adding level to filter: ${checkbox.value}`);
            }
        });
    }
    
    if (selectedLevels.length > 0) {
        params.append('levels', selectedLevels.join(','));
    } else {
        // Default to all levels if none selected
        params.append('levels', 'debug,info,warn,error');
    }
    
    // Add time range
    const timeRange = elements.timeRangeSelect ? elements.timeRangeSelect.value : '24h';
    console.log(`Time range selected: ${timeRange}`);
    
    if (timeRange === 'custom' && elements.startTimeInput && elements.endTimeInput) {
        const startTime = elements.startTimeInput.value;
        const endTime = elements.endTimeInput.value;
        if (startTime) {
            params.append('from', new Date(startTime).toISOString());
        }
        if (endTime) {
            params.append('to', new Date(endTime).toISOString());
        }
    } else {
        // Convert time range to ISO timestamps
        const now = new Date();
        let from;
        
        switch (timeRange) {
            case '15m':
                from = new Date(now.getTime() - 15 * 60 * 1000);
                break;
            case '1h':
                from = new Date(now.getTime() - 60 * 60 * 1000);
                break;
            case '6h':
                from = new Date(now.getTime() - 6 * 60 * 60 * 1000);
                break;
            case '24h':
                from = new Date(now.getTime() - 24 * 60 * 60 * 1000);
                break;
            case '7d':
                from = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
                break;
            case '30d':
                from = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
                break;
            default:
                // Default to 7 days if not specified
                from = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
        }
        
        if (from) {
            params.append('from', from.toISOString());
        }
        params.append('to', now.toISOString());
    }
    
    // Add pagination
    params.append('limit', config.maxLogsPerPage);
    params.append('page', config.currentPage);
    
    return params.toString();
}

// Render logs table
function renderLogs(logs) {
    if (!logs || logs.length === 0) {
        elements.logsTable.innerHTML = '<tr><td colspan="5" class="text-center">No logs found</td></tr>';
        elements.logsPagination.innerHTML = '';
        return;
    }
    
    let html = '';
    logs.forEach(log => {
        const timestamp = new Date(log.timestamp).toLocaleString();
        const level = log.level || 'unknown';
        const source = log.source || 'unknown';
        const message = log.message || 'No message';
        
        html += `
            <tr data-log-id="${log.id || ''}" class="log-row">
                <td>${timestamp}</td>
                <td><span class="level-badge level-${level.toLowerCase()}">${level}</span></td>
                <td><span class="source-badge">${source}</span></td>
                <td class="message-cell">${message}</td>
                <td>
                    <button class="btn btn-sm btn-outline-primary view-log-btn" data-bs-toggle="modal" data-bs-target="#logDetailModal">
                        <i class="fas fa-eye"></i>
                    </button>
                </td>
            </tr>
        `;
    });
    
    elements.logsTable.innerHTML = html;
    
    // Set up event listeners for log detail buttons
    document.querySelectorAll('.view-log-btn').forEach((btn, index) => {
        btn.addEventListener('click', () => {
            showLogDetails(logs[index]);
        });
    });
    
    // Also make the entire row clickable
    document.querySelectorAll('.log-row').forEach((row, index) => {
        row.addEventListener('click', event => {
            // Ensure we're not clicking the button (which has its own handler)
            if (!event.target.closest('.view-log-btn')) {
                showLogDetails(logs[index]);
                elements.logDetailModal.show();
            }
        });
    });
    
    // Update pagination
    renderPagination(logs.length);
}

// Render pagination controls
function renderPagination(logsCount) {
    // Calculate total pages
    config.totalPages = Math.ceil(logsCount / config.maxLogsPerPage) || 1;
    
    let html = '';
    
    // Previous button
    html += `
        <li class="page-item ${config.currentPage === 1 ? 'disabled' : ''}">
            <a class="page-link" href="#" data-page="${config.currentPage - 1}" aria-label="Previous">
                <span aria-hidden="true">&laquo;</span>
            </a>
        </li>
    `;
    
    // Page numbers
    const startPage = Math.max(1, config.currentPage - 2);
    const endPage = Math.min(config.totalPages, config.currentPage + 2);
    
    for (let i = startPage; i <= endPage; i++) {
        html += `
            <li class="page-item ${i === config.currentPage ? 'active' : ''}">
                <a class="page-link" href="#" data-page="${i}">${i}</a>
            </li>
        `;
    }
    
    // Next button
    html += `
        <li class="page-item ${config.currentPage === config.totalPages ? 'disabled' : ''}">
            <a class="page-link" href="#" data-page="${config.currentPage + 1}" aria-label="Next">
                <span aria-hidden="true">&raquo;</span>
            </a>
        </li>
    `;
    
    elements.logsPagination.innerHTML = html;
    
    // Add click events to pagination links
    document.querySelectorAll('.page-link').forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            const page = parseInt(this.dataset.page, 10);
            if (page >= 1 && page <= config.totalPages) {
                config.currentPage = page;
                loadLogs();
            }
        });
    });
}

// Show log details in modal
function showLogDetails(log) {
    elements.detailTimestamp.textContent = new Date(log.timestamp).toLocaleString();
    elements.detailLevel.textContent = log.level || 'unknown';
    elements.detailSource.textContent = log.source || 'unknown';
    elements.detailMessage.textContent = log.message || 'No message';
    
    // Format raw data
    if (log.raw_data) {
        elements.detailRaw.textContent = log.raw_data;
    } else {
        elements.detailRaw.textContent = 'No raw data available';
    }
    
    // Format fields
    if (log.fields && Object.keys(log.fields).length > 0) {
        elements.detailFields.textContent = JSON.stringify(log.fields, null, 2);
    } else {
        elements.detailFields.textContent = 'No fields available';
    }
}

// Update source count badges
function updateSourceCounts(logs) {
    // Reset all counts
    document.querySelectorAll('.source-count').forEach(badge => {
        badge.textContent = '0';
    });
    
    // Count logs by source
    const sourceCounts = {};
    logs.forEach(log => {
        const source = log.source || 'unknown';
        sourceCounts[source] = (sourceCounts[source] || 0) + 1;
    });
    
    // Update badges
    Object.entries(sourceCounts).forEach(([source, count]) => {
        const sourceItem = document.querySelector(`.source-item[data-source="${source}"]`);
        if (sourceItem) {
            const badge = sourceItem.querySelector('.source-count');
            if (badge) {
                badge.textContent = count;
            }
        }
    });
}

// Initialize charts
function initCharts() {
    try {
        console.log("Initializing charts...");
        
        // Volume chart
        if (elements.volumeChartCanvas) {
            const volumeCtx = elements.volumeChartCanvas.getContext('2d');
            config.volumeChart = new Chart(volumeCtx, {
                type: 'line',
                data: {
                    labels: [],
                    datasets: [{
                        label: 'Log Volume',
                        data: [],
                        backgroundColor: config.chartColors.blue,
                        borderColor: 'rgba(54, 162, 235, 1)',
                        borderWidth: 1,
                        tension: 0.3
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    scales: {
                        y: {
                            beginAtZero: true,
                            title: {
                                display: true,
                                text: 'Log Count'
                            }
                        },
                        x: {
                            title: {
                                display: true,
                                text: 'Time'
                            }
                        }
                    }
                }
            });
            console.log("Volume chart initialized");
        } else {
            console.warn("Volume chart canvas element not found");
        }
        
        // Levels chart
        if (elements.levelsChartCanvas) {
            const levelsCtx = elements.levelsChartCanvas.getContext('2d');
            config.levelsChart = new Chart(levelsCtx, {
                type: 'pie',
                data: {
                    labels: [],
                    datasets: [{
                        data: [],
                        backgroundColor: [
                            config.chartColors.grey,   // debug
                            config.chartColors.blue,   // info
                            config.chartColors.yellow, // warn
                            config.chartColors.red     // error
                        ],
                        borderWidth: 1
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: {
                            position: 'right'
                        }
                    }
                }
            });
            console.log("Levels chart initialized");
        } else {
            console.warn("Levels chart canvas element not found");
        }
    } catch (error) {
        console.error("Error initializing charts:", error);
    }
}

// Update charts with new log data
function updateCharts(logs) {
    try {
        if (!logs || !Array.isArray(logs)) {
            console.warn("Invalid logs data for chart update:", logs);
            return;
        }
        
        // Prepare data for volume chart (logs by time)
        const timeGroups = {};
        logs.forEach(log => {
            // Group by hour
            const timestamp = new Date(log.timestamp);
            const hourFormat = timestamp.toLocaleString(undefined, {
                year: 'numeric',
                month: 'numeric',
                day: 'numeric',
                hour: 'numeric'
            });
            
            timeGroups[hourFormat] = (timeGroups[hourFormat] || 0) + 1;
        });
        
        // Sort by time
        const sortedTimes = Object.keys(timeGroups).sort((a, b) => {
            return new Date(a) - new Date(b);
        });
        
        // Update volume chart if it exists
        if (config.volumeChart) {
            config.volumeChart.data.labels = sortedTimes;
            config.volumeChart.data.datasets[0].data = sortedTimes.map(time => timeGroups[time]);
            config.volumeChart.update();
            console.log("Volume chart updated with", sortedTimes.length, "data points");
        }
        
        // Prepare data for levels chart
        const levelCounts = {
            debug: 0,
            info: 0,
            warn: 0,
            error: 0
        };
        
        logs.forEach(log => {
            const level = (log.level || 'unknown').toLowerCase();
            if (levelCounts.hasOwnProperty(level)) {
                levelCounts[level]++;
            }
        });
        
        // Update levels chart if it exists
        if (config.levelsChart) {
            config.levelsChart.data.labels = Object.keys(levelCounts);
            config.levelsChart.data.datasets[0].data = Object.values(levelCounts);
            config.levelsChart.update();
            console.log("Levels chart updated with data:", levelCounts);
        }
    } catch (error) {
        console.error("Error updating charts:", error);
    }
}

// Set up event listeners
function setupEventListeners() {
    try {
        console.log("Setting up event listeners...");
        
        // Search button
        if (elements.searchButton) {
            elements.searchButton.addEventListener('click', () => {
                config.currentPage = 1; // Reset to first page
                loadLogs();
            });
            console.log("Search button event listener added");
        }
        
        // Search input - search on Enter
        if (elements.searchInput) {
            elements.searchInput.addEventListener('keyup', event => {
                if (event.key === 'Enter') {
                    config.currentPage = 1; // Reset to first page
                    loadLogs();
                }
            });
            console.log("Search input event listener added");
        }
        
        // Refresh button
        if (elements.refreshButton) {
            elements.refreshButton.addEventListener('click', () => {
                loadLogs();
            });
            console.log("Refresh button event listener added");
        }
        
        // Auto refresh toggle
        if (elements.autoRefreshButton) {
            elements.autoRefreshButton.addEventListener('click', function() {
                toggleAutoRefresh();
                
                // Toggle active class
                this.classList.toggle('auto-refresh-active');
            });
            console.log("Auto refresh button event listener added");
        }
        
        // Export logs button
        if (elements.exportButton) {
            elements.exportButton.addEventListener('click', () => {
                exportLogs();
            });
            console.log("Export button event listener added");
        }
        
        // Time range select
        if (elements.timeRangeSelect) {
            elements.timeRangeSelect.addEventListener('change', function() {
                // Show/hide custom range inputs
                if (this.value === 'custom' && elements.customRangeDiv) {
                    elements.customRangeDiv.classList.remove('d-none');
                } else if (elements.customRangeDiv) {
                    elements.customRangeDiv.classList.add('d-none');
                    // Reload logs with the new time range
                    loadLogs();
                }
            });
            console.log("Time range select event listener added");
        }
        
        // Custom range inputs
        if (elements.startTimeInput) {
            elements.startTimeInput.addEventListener('change', () => loadLogs());
            console.log("Start time input event listener added");
        }
        
        if (elements.endTimeInput) {
            elements.endTimeInput.addEventListener('change', () => loadLogs());
            console.log("End time input event listener added");
        }
        
        // Level checkboxes
        if (elements.levelCheckboxes) {
            elements.levelCheckboxes.forEach(checkbox => {
                checkbox.addEventListener('change', () => {
                    console.log(`Level checkbox changed: ${checkbox.value} = ${checkbox.checked}`);
                    loadLogs();
                });
            });
            console.log("Level checkbox event listeners added");
        }
        
        // Source item click events already added in renderSources function
        // Make sure they update the active class and trigger loadLogs
        document.querySelectorAll('.source-item').forEach(sourceItem => {
            sourceItem.addEventListener('click', (e) => {
                e.preventDefault();
                const source = sourceItem.dataset.source;
                console.log(`Source item clicked: ${source}`);
                // Toggle active class
                sourceItem.classList.toggle('active');
                // Reload logs
                loadLogs();
            });
        });
        console.log("Source item event listeners added");
        
        console.log("All event listeners set up successfully");
    } catch (error) {
        console.error("Error setting up event listeners:", error);
    }
}

// Toggle auto refresh
function toggleAutoRefresh() {
    config.autoRefreshEnabled = !config.autoRefreshEnabled;
    
    if (config.autoRefreshEnabled) {
        // Start periodic refresh
        config.refreshTimer = setInterval(() => {
            loadLogs();
        }, config.refreshInterval);
        
        // Update button text
        elements.autoRefreshButton.innerHTML = '<i class="fas fa-clock"></i> Auto Refreshing...';
    } else {
        // Stop periodic refresh
        clearInterval(config.refreshTimer);
        
        // Reset button text
        elements.autoRefreshButton.innerHTML = '<i class="fas fa-clock"></i> Auto Refresh';
    }
}

// Export logs to file
function exportLogs() {
    // Build query for export (without pagination)
    const params = buildQueryParams();
    const url = `${config.apiBaseUrl}/logs/export?${params}`;
    
    fetch(url)
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error! Status: ${response.status}`);
            }
            return response.json();
        })
        .then(logs => {
            // Convert to CSV
            const csv = convertLogsToCSV(logs);
            
            // Create download link
            const blob = new Blob([csv], { type: 'text/csv' });
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.setAttribute('hidden', '');
            a.setAttribute('href', url);
            a.setAttribute('download', `logstream_export_${new Date().toISOString()}.csv`);
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
        })
        .catch(error => {
            console.error('Error exporting logs:', error);
            showErrorMessage('Failed to export logs. Please try again.');
        });
}

// Convert logs to CSV format
function convertLogsToCSV(logs) {
    if (!logs || logs.length === 0) {
        return 'No logs to export';
    }
    
    // CSV header
    const header = ['Timestamp', 'Level', 'Source', 'Message', 'Fields'];
    
    // CSV rows
    const rows = logs.map(log => {
        return [
            new Date(log.timestamp).toISOString(),
            log.level || '',
            log.source || '',
            log.message || '',
            log.fields ? JSON.stringify(log.fields) : ''
        ].map(field => {
            // Escape double quotes and wrap in quotes
            return `"${String(field).replace(/"/g, '""')}"`;
        }).join(',');
    });
    
    // Combine header and rows
    return [header.join(','), ...rows].join('\n');
}

// Show loading indicator
function showLoading() {
    // Add loading indicator to logs table
    if (elements.logsTable.querySelector('tr td.loading-cell') === null) {
        const loadingRow = document.createElement('tr');
        loadingRow.innerHTML = `
            <td colspan="5" class="text-center loading-cell">
                <div class="loader"></div> Loading logs...
            </td>
        `;
        elements.logsTable.appendChild(loadingRow);
    }
}

// Hide loading indicator
function hideLoading() {
    // Remove loading indicators
    const loadingCells = elements.logsTable.querySelectorAll('td.loading-cell');
    loadingCells.forEach(cell => {
        cell.parentElement.remove();
    });
}

// Show error message
function showErrorMessage(message) {
    // Create alert
    const alert = document.createElement('div');
    alert.className = 'alert alert-danger alert-dismissible fade show mt-2';
    alert.innerHTML = `
        ${message}
        <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
    `;
    
    // Add to the top of the log viewer card
    const cardBody = document.querySelector('.card-body');
    if (cardBody) {
        cardBody.insertBefore(alert, cardBody.firstChild);
        
        // Auto-remove after 5 seconds
        setTimeout(() => {
            alert.remove();
        }, 5000);
    }
}