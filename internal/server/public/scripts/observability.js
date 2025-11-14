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

    var errorCharts = {
        page: new Chart(document.getElementById('errorsPageChart').getContext('2d'), {
            type: 'line',
            plugins: [backgroundColorPlugin, ChartDatasourcePrometheusPlugin],
            options: {
                ...commonOptions,
                plugins: {
                    ...commonOptions.plugins,
                    title: {
                        display: true,
                        text: 'Errors - Page Request Failures (Max Latency)',
                        color: '#FFF',
                        font: { size: 14 }
                    },
                    'datasource-prometheus': {
                        prometheus: {
                            endpoint: "http://localhost:8080",
                            baseURL: "/metrics",
                        },
                        query: 'max by (http.route) (http.server.request.duration{http.route!~"/api.*"})',
                        timeRange: timeRange
                    }
                },
                scales: {
                    ...commonOptions.scales,
                    y: {
                        ...commonOptions.scales.y,
                        ticks: {
                            ...commonOptions.scales.y.ticks,
                            callback: function (value) {
                                return (value * 1000).toFixed(2) + ' ms';
                            }
                        }
                    }
                }
            }
        }),
        api: new Chart(document.getElementById('errorsApiChart').getContext('2d'), {
            type: 'line',
            plugins: [backgroundColorPlugin, ChartDatasourcePrometheusPlugin],
            options: {
                ...commonOptions,
                plugins: {
                    ...commonOptions.plugins,
                    title: {
                        display: true,
                        text: 'Errors - API Request Failures (Max Latency)',
                        color: '#FFF',
                        font: { size: 14 }
                    },
                    'datasource-prometheus': {
                        prometheus: {
                            endpoint: "http://localhost:8080",
                            baseURL: "/metrics",
                        },
                        query: 'max by (http.route) (http.server.request.duration{http.route=~"/api.*"})',
                        timeRange: timeRange
                    }
                },
                scales: {
                    ...commonOptions.scales,
                    y: {
                        ...commonOptions.scales.y,
                        ticks: {
                            ...commonOptions.scales.y.ticks,
                            callback: function (value) {
                                return (value * 1000).toFixed(2) + ' ms';
                            }
                        }
                    }
                }
            }
        })
    };

    var latencyCharts = {
        page: new Chart(document.getElementById('latencyPageChart').getContext('2d'), {
            type: 'line',
            plugins: [backgroundColorPlugin, ChartDatasourcePrometheusPlugin],
            options: {
                ...commonOptions,
                plugins: {
                    ...commonOptions.plugins,
                    title: {
                        display: true,
                        text: 'Latency - Slow Page Requests (>1s)',
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
        }),
        pageP99: new Chart(document.getElementById('latencyPageP99Chart').getContext('2d'), {
            type: 'line',
            plugins: [backgroundColorPlugin, ChartDatasourcePrometheusPlugin],
            options: {
                ...commonOptions,
                plugins: {
                    ...commonOptions.plugins,
                    title: {
                        display: true,
                        text: 'Latency - Page Request Duration (p99)',
                        color: '#FFF',
                        font: { size: 14 }
                    },
                    'datasource-prometheus': {
                        prometheus: {
                            endpoint: "http://localhost:8080",
                            baseURL: "/metrics",
                        },
                        query: 'max by (http.route) (http.server.request.duration{http.route!~"/api.*"})',
                        timeRange: timeRange
                    }
                },
                scales: {
                    ...commonOptions.scales,
                    y: {
                        ...commonOptions.scales.y,
                        ticks: {
                            ...commonOptions.scales.y.ticks,
                            callback: function (value) {
                                return (value * 1000).toFixed(2) + ' ms';
                            }
                        }
                    }
                }
            }
        }),
        api: new Chart(document.getElementById('latencyApiChart').getContext('2d'), {
            type: 'line',
            plugins: [backgroundColorPlugin, ChartDatasourcePrometheusPlugin],
            options: {
                ...commonOptions,
                plugins: {
                    ...commonOptions.plugins,
                    title: {
                        display: true,
                        text: 'Latency - Slow API Requests (>1s)',
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
        }),
        apiP99: new Chart(document.getElementById('latencyApiP99Chart').getContext('2d'), {
            type: 'line',
            plugins: [backgroundColorPlugin, ChartDatasourcePrometheusPlugin],
            options: {
                ...commonOptions,
                plugins: {
                    ...commonOptions.plugins,
                    title: {
                        display: true,
                        text: 'Latency - API Request Duration (p99)',
                        color: '#FFF',
                        font: { size: 14 }
                    },
                    'datasource-prometheus': {
                        prometheus: {
                            endpoint: "http://localhost:8080",
                            baseURL: "/metrics",
                        },
                        query: 'max by (http.route) (http.server.request.duration{http.route=~"/api.*"})',
                        timeRange: timeRange
                    }
                },
                scales: {
                    ...commonOptions.scales,
                    y: {
                        ...commonOptions.scales.y,
                        ticks: {
                            ...commonOptions.scales.y.ticks,
                            callback: function (value) {
                                return (value * 1000).toFixed(2) + ' ms';
                            }
                        }
                    }
                }
            }
        })
    };

    var trafficCharts = {
        page: new Chart(document.getElementById('trafficPageChart').getContext('2d'), {
            type: 'line',
            plugins: [backgroundColorPlugin, ChartDatasourcePrometheusPlugin],
            options: {
                ...commonOptions,
                plugins: {
                    ...commonOptions.plugins,
                    title: {
                        display: true,
                        text: 'Traffic - Page Request Count',
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
        }),
        api: new Chart(document.getElementById('trafficApiChart').getContext('2d'), {
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
        })
    };

    var saturationCharts = {
        memory: new Chart(document.getElementById('saturationChart').getContext('2d'), {
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
                            callback: function (value) {
                                return (value / 1024 / 1024).toFixed(2) + ' MB';
                            }
                        }
                    }
                }
            }
        })
    };
}

function initializeHealthTabs() {
    document.querySelectorAll('.tab-button').forEach(button => {
        button.addEventListener('click', function () {
            const tabName = this.getAttribute('data-tab');

            // Update button styles
            document.querySelectorAll('.tab-button').forEach(btn => {
                btn.classList.remove('active');
            });
            this.classList.add('active');

            // Show/hide content
            document.querySelectorAll('.tab-content').forEach(content => {
                content.classList.remove('active');
            });
            document.querySelector(`.tab-content[data-content="${tabName}"]`).classList.add('active');
        });
    });
}
