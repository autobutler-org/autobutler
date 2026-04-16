class DataCell<T> {
  T value;

  DataCell(this.value);
  DataCell.unnamed(this.value);

  String toJson() => value.toString();

  static DataCell<String> fromJson(dynamic json) =>
      DataCell<String>(json?.toString() ?? '');
}
