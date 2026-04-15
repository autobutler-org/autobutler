import 'package:flutter/material.dart'
    show
        StatelessWidget,
        Widget,
        BuildContext,
        TextEditingController,
        VoidCallback,
        PointerDownEvent,
        InputDecoration,
        EdgeInsets,
        InputBorder,
        TextAlignVertical,
        TextField;

class EditableCell extends StatelessWidget {
  final TextEditingController controller;
  final void Function(String) onSubmitted;
  final VoidCallback onEditingComplete;
  final void Function(PointerDownEvent) onTapOutside;

  const EditableCell({
    super.key,
    required this.controller,
    required this.onSubmitted,
    required this.onEditingComplete,
    required this.onTapOutside,
  });

  @override
  Widget build(BuildContext context) {
    return TextField(
      autofocus: true,
      controller: controller,
      decoration: const InputDecoration(
        contentPadding: EdgeInsets.symmetric(horizontal: 8, vertical: 8),
        isDense: true,
        border: InputBorder.none,
      ),
      textAlignVertical: TextAlignVertical.center,
      onSubmitted: onSubmitted,
      onEditingComplete: onEditingComplete,
      onTapOutside: onTapOutside,
    );
  }
}
