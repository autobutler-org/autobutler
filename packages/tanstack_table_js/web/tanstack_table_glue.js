(function(window){
  // Minimal JS glue that renders a simple HTML table. Designed to be replaced by a
  // full @tanstack/table-core renderer later while keeping the same external API.

  function ensureContainer(elementId){
    var el = document.getElementById(elementId);
    if(!el){
      // If the element isn't found, try to find the platform view container created by Flutter.
      // Flutter's HtmlElementView creates a div with a shadow element; the provided view id should be present.
      el = document.querySelector('#' + elementId);
    }
    return el;
  }

  function buildTable(el, data, columns){
    el.innerHTML = '';
    var table = document.createElement('table');
    table.style.borderCollapse = 'collapse';
    table.style.width = '100%';
    var thead = document.createElement('thead');
    var tr = document.createElement('tr');
    columns.forEach(function(col){
      var th = document.createElement('th');
      th.textContent = col && col.header ? col.header : (col && col.accessor ? col.accessor : '');
      th.style.border = '1px solid #ccc';
      th.style.padding = '6px';
      tr.appendChild(th);
    });
    thead.appendChild(tr);
    table.appendChild(thead);

    var tbody = document.createElement('tbody');
    (data || []).forEach(function(row){
      var trr = document.createElement('tr');
      columns.forEach(function(col){
        var td = document.createElement('td');
        var val = '';
        try{
          val = row[col.accessor];
        }catch(e){ val = '' }
        td.textContent = (val === undefined || val === null) ? '' : String(val);
        td.style.border = '1px solid #eee';
        td.style.padding = '6px';
        trr.appendChild(td);
      });
      tbody.appendChild(trr);
    });
    table.appendChild(tbody);
    el.appendChild(table);
  }

  window.createTanstackTable = function(elementId, data, columns){
    var el = ensureContainer(elementId);
    if(!el){
      console.warn('createTanstackTable: element not found:', elementId);
      return;
    }
    buildTable(el, data, columns);
    // store current state
    el.__tanstack_table_state = { data: data, columns: columns };
  };

  window.updateTanstackTable = function(elementId, data, columns){
    var el = ensureContainer(elementId);
    if(!el){
      console.warn('updateTanstackTable: element not found:', elementId);
      return;
    }
    buildTable(el, data, columns);
    el.__tanstack_table_state = { data: data, columns: columns };
  };

  // Expose a simple API namespace as well
  window.tanstack_table = {
    create: window.createTanstackTable,
    update: window.updateTanstackTable
  };

})(window);
