function createGoldenSignalCharts() {
    Chart.registry.plugins.register(ChartDatasourcePrometheusPlugin);
    Chart.defaults.color = '#FFF';

    var backgroundColorPlugin = {
        id: 'customCanvasBackgroundColor',
        beforeDraw: (chart) => {
            const ctx = chart.ctx;
            ctx.fillStyle = 'rgba(128, 128, 128, 0.1)';
            ctx.fillRect(0, 0, chart.width, chart.height);
        }
    };

    const commonOptions = {
        responsive: true,
        maintainAspectRatio: true,
        aspectRatio: 4,
        plugins: {
            legend: {
                display: true,
                labels: {
                    color: '#FFF',
                    font: { size: 11 }
                }
            }
        },
        scales: {
            y: {
                ticks: { color: '#FFF', font: { size: 10 } },
                grid: { color: 'rgba(255, 255, 255, 0.1)' }
            },
            x: {
                ticks: { color: '#FFF', font: { size: 10 } },
                grid: { color: 'rgba(255, 255, 255, 0.1)' }
            }
        }
    };

    const timeRange = {
        type: 'relative',
        start: -1 * 60 * 60 * 1000,
        end: 0
    };

    // Golden Signal 1: Page Latency
    var latencyPageChart = new Chart(document.getElementById('latencyPageChart').getContext('2d'), {
        type: 'line',
        plugins: [backgroundColorPlugin, ChartDatasourcePrometheusPlugin],
        options: {
            ...commonOptions,
            plugins: {
                ...commonOptions.plugins,
                title: {
                    display: true,
                    text: 'Latency - Page Request Duration (Non-API)',
                    color: '#FFF',
                    font: { size: 14 }
                },
                'datasource-prometheus': {
                    prometheus: {
                        endpoint: "http://localhost:8080",
                        baseURL: "/metrics",
                    },
                    query: 'avg by (http.route) (http.server.request.duration{http.route!~"/api.*"})',
                    timeRange: timeRange
                }
            },
            scales: {
                ...commonOptions.scales,
                y: {
                    ...commonOptions.scales.y,
                    ticks: {
                        ...commonOptions.scales.y.ticks,
                        callback: function(value) {
                            return (value * 1000).toFixed(2) + ' ms';
                        }
                    }
                }
            }
        }
    });

    // Golden Signal 2: API Latency
    var latencyApiChart = new Chart(document.getElementById('latencyApiChart').getContext('2d'), {
        type: 'line',
        plugins: [backgroundColorPlugin, ChartDatasourcePrometheusPlugin],
        options: {
            ...commonOptions,
            plugins: {
                ...commonOptions.plugins,
                title: {
                    display: true,
                    text: 'Latency - API Request Duration',
                    color: '#FFF',
                    font: { size: 14 }
                },
                'datasource-prometheus': {
                    prometheus: {
                        endpoint: "http://localhost:8080",
                        baseURL: "/metrics",
                    },
                    query: 'avg by (http.route) (http.server.request.duration{http.route=~"/api.*"})',
                    timeRange: timeRange
                }
            },
            scales: {
                ...commonOptions.scales,
                y: {
                    ...commonOptions.scales.y,
                    ticks: {
                        ...commonOptions.scales.y.ticks,
                        callback: function(value) {
                            return (value * 1000).toFixed(2) + ' ms';
                        }
                    }
                }
            }
        }
    });

    // Golden Signal 3: Traffic (Non-API Routes)
    var trafficPageChart = new Chart(document.getElementById('trafficPageChart').getContext('2d'), {
        type: 'line',
        plugins: [backgroundColorPlugin, ChartDatasourcePrometheusPlugin],
        options: {
            ...commonOptions,
            plugins: {
                ...commonOptions.plugins,
                title: {
                    display: true,
                    text: 'Traffic - Page Request Count (Non-API)',
                    color: '#FFF',
                    font: { size: 14 }
                },
                'datasource-prometheus': {
                    prometheus: {
                        endpoint: "http://localhost:8080",
                        baseURL: "/metrics",
                    },
                    query: 'count by (http.route) (http.server.request.duration{http.route!~"/api.*"})',
                    timeRange: timeRange
                }
            }
        }
    });

    // Golden Signal 4: API Traffic
    var trafficApiChart = new Chart(document.getElementById('trafficApiChart').getContext('2d'), {
        type: 'line',
        plugins: [backgroundColorPlugin, ChartDatasourcePrometheusPlugin],
        options: {
            ...commonOptions,
            plugins: {
                ...commonOptions.plugins,
                title: {
                    display: true,
                    text: 'Traffic - API Request Count',
                    color: '#FFF',
                    font: { size: 14 }
                },
                'datasource-prometheus': {
                    prometheus: {
                        endpoint: "http://localhost:8080",
                        baseURL: "/metrics",
                    },
                    query: 'count by (http.route) (http.server.request.duration{http.route=~"/api.*"})',
                    timeRange: timeRange
                }
            }
        }
    });

    // Golden Signal 4: Saturation
    var saturationChart = new Chart(document.getElementById('saturationChart').getContext('2d'), {
        type: 'line',
        plugins: [backgroundColorPlugin, ChartDatasourcePrometheusPlugin],
        options: {
            ...commonOptions,
            plugins: {
                ...commonOptions.plugins,
                title: {
                    display: true,
                    text: 'Saturation - Memory Usage',
                    color: '#FFF',
                    font: { size: 14 }
                },
                'datasource-prometheus': {
                    prometheus: {
                        endpoint: "http://localhost:8080",
                        baseURL: "/metrics",
                    },
                    query: 'go.memory.allocated',
                    timeRange: timeRange
                }
            },
            scales: {
                ...commonOptions.scales,
                y: {
                    ...commonOptions.scales.y,
                    ticks: {
                        ...commonOptions.scales.y.ticks,
                        callback: function(value) {
                            return (value / 1024 / 1024).toFixed(2) + ' MB';
                        }
                    }
                }
            }
        }
    });
}
